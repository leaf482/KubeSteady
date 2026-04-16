# KubeSteady

**A deterministic, fail-safe SRE control loop for Kubernetes resource optimization.**

## Overview

KubeSteady is a production-style control loop that ingests operational metrics, stabilizes noisy signals, generates deterministic scaling recommendations, and exposes the full decision path through an API and dashboard.

It is designed for SRE-focused reliability engineering: predictable behavior, explicit safety guardrails, and graceful degradation when telemetry is missing or unstable.

## Demo / Screenshot

> Add dashboard screenshot/GIF here (ControlStatus, ResourceCore, MetricsPanel, PodTable).
>
> Example:
> `![KubeSteady Dashboard](./docs/demo-dashboard.png)`

## Key Features

- Deterministic recommendation pipeline (no AI, no randomness)
- Prometheus metric ingestion with fail-safe deterministic mock fallback
- Sliding window aggregation (5m) + EMA smoothing for stable decision signals
- Confidence scoring from signal variance (stability-aware)
- Safety validation layer and cooldown-based anti-flapping protection
- Rollback signal evaluator based on pre/post metric comparison
- Snapshot API (`/snapshot`) for full system-state observability
- React dashboard with live polling and source visibility (`PROMETHEUS` / `MOCK`)
- Metric-agnostic signal handling (CPU-style and latency-style metrics)
- Latency-aware override logic without changing core pipeline structure

## Architecture

KubeSteady separates data collection, signal processing, decisioning, and observability into explicit deterministic stages.

```text
Prometheus / Blackbox Exporter
            |
            v
        Collector
            |
            v
       Aggregator (5m window)
            |
            v
      Smoother (EMA)
            |
            v
       Recommender
            |
            v
        Validator
            |
            v
        Cooldown
            |
            v
        Evaluator
            |
            v
      Snapshot Store
            |
            v
   REST API (/snapshot) ---> React Dashboard
```

## How It Works (Control Loop)

Each tick (`poll_interval`) executes a fixed deterministic pipeline:

1. **Collect**  
   Query Prometheus for the configured signal (CPU or latency).

2. **Aggregate**  
   Append samples to per-target in-memory windows and evict entries older than 5 minutes.

3. **Smooth**  
   Apply EMA to reduce short-term noise.

4. **Recommend**  
   Generate `scale_up` / `scale_down` / `no_op` decisions based on deterministic thresholds.

5. **Validate**  
   Apply confidence and safety guardrails to prevent unsafe actions.

6. **Cooldown**  
   Suppress rapid repeated actions to prevent flapping.

7. **Evaluate**  
   Compare pre/post smoothed signals and emit rollback intent signals (read-only).

## Fail-Safe Design

KubeSteady is intentionally fail-safe:

- If Prometheus is unavailable, invalid, or returns empty results, collector returns deterministic mock metrics.
- The control loop continues running instead of failing closed.
- This ensures API/dashboard continuity and enables testability even without live telemetry.

Deterministic fallback values:

- `pod-a: 0.62`
- `pod-b: 0.81`
- `pod-c: 0.21`

## Metric-Agnostic Design

The pipeline processes a generic numeric signal (`PodCPUUsage` as a historical naming artifact).

- **CPU mode:** classic thresholds (`>0.75`, `<0.25`)
- **Latency mode:** enabled when source is Prometheus and query includes `probe_duration_seconds`  
  - high threshold: `0.5s`
  - low threshold: `0.2s`

Blackbox Exporter integration enables external endpoint monitoring (latency/probe metrics) using the same control-loop infrastructure.

## Tech Stack

- **Backend:** Go
- **Metrics:** Prometheus
- **External probing:** Blackbox Exporter
- **Frontend:** React + Vite + Tailwind CSS
- **API:** REST (`/snapshot`)

## Getting Started

### One-Command Local Control (Recommended)

Use the root `Makefile` to start, stop, and restart the full local test stack:

```powershell
make up
make down
make restart
make status
```

What `make up` starts:

- Prometheus + Blackbox Exporter (`monitoring/docker-compose.yml`)
- Go backend control loop
- React frontend dev server

Useful overrides:

```powershell
make up PROMETHEUS_URL=http://127.0.0.1:9090
make up PROMETHEUS_QUERY='avg_over_time(probe_duration_seconds{job="blackbox-http"}[5m])'
make up POLL_INTERVAL=10s LOG_LEVEL=debug
```

Logs are written to:

- `logs/backend.log`
- `logs/backend.err.log`
- `logs/frontend.log`
- `logs/frontend.err.log`

> Windows note: install GNU Make first (e.g., via Chocolatey: `choco install make`).

### 1) Run Backend

```powershell
cd D:\Github\KubeSteady
go run .\cmd\server\main.go
```

Optional environment variables:

```powershell
$env:PROMETHEUS_URL="http://127.0.0.1:9090"
$env:PROMETHEUS_QUERY="sum(rate(container_cpu_usage_seconds_total{pod!=""}[5m])) by (pod)"
$env:POLL_INTERVAL="30s"
$env:LOG_LEVEL="info"
```

### 2) Run Frontend

```powershell
cd D:\Github\KubeSteady\frontend
npm install
npm run dev
```

Frontend default URL: `http://localhost:5173`  
Backend snapshot API: `http://localhost:8080/snapshot`

### 3) (Optional) Run Prometheus + Blackbox Exporter

```powershell
cd D:\Github\KubeSteady\monitoring
docker compose up -d
```

Run backend with latency query:

```powershell
cd D:\Github\KubeSteady
$env:PROMETHEUS_URL="http://127.0.0.1:9090"
$env:PROMETHEUS_QUERY="avg_over_time(probe_duration_seconds{job=""blackbox-http""}[5m])"
go run .\cmd\server\main.go
```

Check targets: `http://127.0.0.1:9090/targets`

## Example Use Case

Monitor public endpoints such as:

- `https://www.google.com`
- `https://open.spotify.com/`

with Blackbox Exporter, then drive deterministic actions from latency trends:

- high sustained latency -> `scale_up`
- healthy/stable latency -> `no_op`

All decisions, confidence, validation outcomes, and rollback signals are visible in `/snapshot` and the dashboard.

## Future Improvements

- Kubernetes execution layer (apply validated recommendations safely)
- Multi-metric policy engine (latency + error rate + saturation)
- Persistent state for cooldown/history across restarts
- Policy/config profiles per workload class
- Alerting hooks for rollback triggers and validation failures

## Author / Closing

KubeSteady is built as a practical control-loop architecture project for engineers focused on SRE, platform reliability, and safe automation design.

If you are evaluating this for engineering roles, focus on the deterministic pipeline design, guardrail-first decisioning, and observability-first implementation approach.
