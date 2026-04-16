package main

import (
	"os"

	"kubesteady/internal/config"
	"kubesteady/internal/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel)

	logger.Info("server starting")
	logger.Info("configuration loaded", "prometheus_url", cfg.PrometheusURL, "poll_interval", cfg.PollInterval.String())

	// Skeleton lifecycle only: start and clean exit.
	logger.Info("server stopped")
	os.Exit(0)
}
