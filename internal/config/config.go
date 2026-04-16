package config

import (
	"os"
	"time"
)

const (
	defaultPrometheusURL = "http://localhost:9090"
	defaultPrometheusQuery = `sum(rate(container_cpu_usage_seconds_total{pod!=""}[5m])) by (pod)`
	defaultPollInterval  = 30 * time.Second
	defaultLogLevel      = "info"
)

type Config struct {
	PrometheusURL string
	PrometheusQuery string
	PollInterval  time.Duration
	LogLevel      string
}

func Load() Config {
	cfg := Config{
		PrometheusURL: getEnv("PROMETHEUS_URL", defaultPrometheusURL),
		PrometheusQuery: getEnv("PROMETHEUS_QUERY", defaultPrometheusQuery),
		LogLevel:      getEnv("LOG_LEVEL", defaultLogLevel),
		PollInterval:  defaultPollInterval,
	}

	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.PollInterval = d
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
