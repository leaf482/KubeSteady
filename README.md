# KubeSteady
A High-Reliability Kubernetes Resource Optimizer based on SRE Principles

## Minimal Skeleton

This repository currently contains a minimal compile-safe Go project skeleton:
- `cmd/server/main.go`: entrypoint that loads config, initializes logging, and exits cleanly.
- `internal/config`: env-based configuration (`PROMETHEUS_URL`, `POLL_INTERVAL`, `LOG_LEVEL`).
- `internal/logging`: health-oriented structured logger setup.
- `internal/metrics`, `internal/analyzer`: interfaces/placeholders only.
- `internal/optimizer/types.go`: shared core optimization type placeholder.

## Blackbox Exporter Integration

You can collect public website probe metrics and feed them into KubeSteady using:
- `monitoring/docker-compose.yml`
- `monitoring/prometheus.yml`
- `monitoring/blackbox.yml`

### 1) Start monitoring stack

```powershell
cd D:\Github\KubeSteady\monitoring
docker compose up -d
```

### 2) Run backend with blackbox query

```powershell
cd D:\Github\KubeSteady
$env:PROMETHEUS_URL="http://127.0.0.1:9090"
$env:PROMETHEUS_QUERY="avg_over_time(probe_duration_seconds{job=""blackbox-http""}[5m]) by (instance)"
go run .\cmd\server\main.go
```

### 3) Run frontend

```powershell
cd D:\Github\KubeSteady\frontend
npm run dev
```

If Prometheus/blackbox is unavailable or returns empty data, collector falls back to deterministic mock values.
