package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kubesteady/internal/config"
	"kubesteady/internal/logging"
	"kubesteady/internal/metrics"
	"kubesteady/internal/observability"
	"kubesteady/internal/optimizer"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	collector := metrics.NewPrometheusCollector(cfg)
	aggregator := metrics.NewAggregator(0)
	smoother := metrics.NewSmoother(0)
	recommender := optimizer.Recommender{}
	validator := optimizer.Validator{}
	cooldown := optimizer.NewCooldownManager(0)
	evaluator := optimizer.Evaluator{}
	snapshots := &observability.SnapshotStore{}

	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot", handleSnapshot(snapshots))
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	var previousSmoothed []metrics.SmoothedCPUUsage

	logger.Info("server starting")
	logger.Info("configuration loaded", "prometheus_url", cfg.PrometheusURL, "poll_interval", cfg.PollInterval.String())
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown requested")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("http shutdown failed", "error", err)
			}
			logger.Info("server stopped")
			return
		case <-ticker.C:
			logger.Info("tick start")

			podCPU, err := collector.Collect(ctx)
			if err != nil {
				logger.Error("collect failed", "error", err)
				continue
			}

			windowed := aggregator.Aggregate(podCPU)
			smoothed := smoother.Smooth(windowed)
			recommender.LatencyMode = collector.DataSource() == "prometheus" && strings.Contains(strings.ToLower(cfg.PrometheusQuery), "probe_duration_seconds")
			recs := recommender.Recommend(smoothed, aggregator)
			validated := validator.Validate(recs)
			cooled := cooldown.ApplyCooldown(validated)

			evals := make([]optimizer.EvaluationResult, 0)
			rollbackTriggers := 0
			if len(previousSmoothed) > 0 {
				evals = evaluator.Evaluate(previousSmoothed, smoothed)
				for _, eval := range evals {
					if eval.ShouldRollback {
						rollbackTriggers++
					}
				}
			}

			scaleUp := 0
			scaleDown := 0
			noOp := 0
			for _, rec := range cooled {
				switch rec.Action {
				case "scale_up":
					scaleUp++
				case "scale_down":
					scaleDown++
				default:
					noOp++
				}
			}

			var totalCPU float64
			highCPUCount := 0
			lowCPUCount := 0
			for _, usage := range smoothed {
				totalCPU += usage.CPU
				if usage.CPU > 0.75 {
					highCPUCount++
				}
				if usage.CPU < 0.25 {
					lowCPUCount++
				}
			}

			avgCPU := 0.0
			if len(smoothed) > 0 {
				avgCPU = totalCPU / float64(len(smoothed))
			}

			snapshots.Update(observability.SystemSnapshot{
				Timestamp:       time.Now(),
				Pods:            len(smoothed),
				DataSource:      collector.DataSource(),
				SmoothedCPU:     smoothed,
				Recommendations: recs,
				Validated:       cooled,
				Rollbacks:       evals,
			})

			logger.Info(
				"tick complete",
				"pods_processed", len(smoothed),
				"avg_cpu", avgCPU,
				"high_cpu_pods", highCPUCount,
				"low_cpu_pods", lowCPUCount,
				"scale_up", scaleUp,
				"scale_down", scaleDown,
				"no_op", noOp,
				"rollback_triggers", rollbackTriggers,
			)

			previousSmoothed = append(previousSmoothed[:0], smoothed...)
		}
	}

}

func handleSnapshot(store *observability.SnapshotStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		snapshot := store.Get()
		if snapshot.Timestamp.IsZero() {
			snapshot.Timestamp = time.Now()
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			http.Error(w, "failed to encode snapshot", http.StatusInternalServerError)
			return
		}
	}
}
