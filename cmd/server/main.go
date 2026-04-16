package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"kubesteady/internal/config"
	"kubesteady/internal/logging"
	"kubesteady/internal/metrics"
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

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	var previousSmoothed []metrics.SmoothedCPUUsage

	logger.Info("server starting")
	logger.Info("configuration loaded", "prometheus_url", cfg.PrometheusURL, "poll_interval", cfg.PollInterval.String())

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown requested")
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
			recs := recommender.Recommend(smoothed, aggregator)
			validated := validator.Validate(recs)
			cooled := cooldown.ApplyCooldown(validated)

			rollbackTriggers := 0
			if len(previousSmoothed) > 0 {
				evals := evaluator.Evaluate(previousSmoothed, smoothed)
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

			logger.Info(
				"tick complete",
				"pods_processed", len(smoothed),
				"scale_up", scaleUp,
				"scale_down", scaleDown,
				"no_op", noOp,
				"rollback_triggers", rollbackTriggers,
			)

			previousSmoothed = append(previousSmoothed[:0], smoothed...)
		}
	}

}
