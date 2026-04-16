BACKEND_PID_FILE := .backend.pid
FRONTEND_PID_FILE := .frontend.pid
LOG_DIR := logs

PROMETHEUS_URL ?= http://127.0.0.1:9090
PROMETHEUS_QUERY ?= avg_over_time(probe_duration_seconds[5m])
POLL_INTERVAL ?= 30s
LOG_LEVEL ?= info

BACKEND_OUT_LOG_FILE := $(LOG_DIR)/backend.log
BACKEND_ERR_LOG_FILE := $(LOG_DIR)/backend.err.log
FRONTEND_OUT_LOG_FILE := $(LOG_DIR)/frontend.log
FRONTEND_ERR_LOG_FILE := $(LOG_DIR)/frontend.err.log

.PHONY: up down restart status monitoring-up monitoring-down backend-up backend-down frontend-up frontend-down

up: monitoring-up backend-up frontend-up status

down: backend-down frontend-down monitoring-down status

restart: down up

status:
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$targetPid = ''; if (Test-Path '$(BACKEND_PID_FILE)') { $$targetPid = (Get-Content '$(BACKEND_PID_FILE)' | Select-Object -First 1).Trim() }; if ($$targetPid -and (Get-Process -Id $$targetPid -ErrorAction SilentlyContinue)) { Write-Host \"backend: running (PID=$$targetPid)\" } else { Write-Host 'backend: stopped' }"
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$targetPid = ''; if (Test-Path '$(FRONTEND_PID_FILE)') { $$targetPid = (Get-Content '$(FRONTEND_PID_FILE)' | Select-Object -First 1).Trim() }; if ($$targetPid -and (Get-Process -Id $$targetPid -ErrorAction SilentlyContinue)) { Write-Host \"frontend: running (PID=$$targetPid)\" } else { Write-Host 'frontend: stopped' }"
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "docker ps --format 'table {{.Names}}\t{{.Status}}' | Select-String 'kubesteady-prometheus|kubesteady-blackbox|NAMES'"

monitoring-up:
	@docker compose -f monitoring/docker-compose.yml up -d

monitoring-down:
	@docker compose -f monitoring/docker-compose.yml down

backend-up:
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$targetPid = ''; if (Test-Path '$(BACKEND_PID_FILE)') { $$targetPid = (Get-Content '$(BACKEND_PID_FILE)' | Select-Object -First 1).Trim() }; if ($$targetPid -and (Get-Process -Id $$targetPid -ErrorAction SilentlyContinue)) { Write-Host \"backend already running (PID=$$targetPid)\"; exit 0 }; New-Item -ItemType Directory -Path '$(LOG_DIR)' -Force | Out-Null; $$cmd = '$$env:PROMETHEUS_URL=''$(PROMETHEUS_URL)''; $$env:PROMETHEUS_QUERY=''$(PROMETHEUS_QUERY)''; $$env:POLL_INTERVAL=''$(POLL_INTERVAL)''; $$env:LOG_LEVEL=''$(LOG_LEVEL)''; go run .\cmd\server\main.go'; $$proc = Start-Process powershell -ArgumentList \"-NoProfile -ExecutionPolicy Bypass -Command $$cmd\" -WorkingDirectory \"$(CURDIR)\" -RedirectStandardOutput \"$(BACKEND_OUT_LOG_FILE)\" -RedirectStandardError \"$(BACKEND_ERR_LOG_FILE)\" -PassThru; if (-not $$proc) { throw 'failed to start backend process' }; $$proc.Id | Out-File '$(BACKEND_PID_FILE)' -Encoding ascii; Write-Host \"backend started (PID=$$($$proc.Id))\""

backend-down:
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$targetPid = ''; if (Test-Path '$(BACKEND_PID_FILE)') { $$targetPid = (Get-Content '$(BACKEND_PID_FILE)' | Select-Object -First 1).Trim() }; if ($$targetPid -and (Get-Process -Id $$targetPid -ErrorAction SilentlyContinue)) { Stop-Process -Id $$targetPid -Force; Write-Host \"backend stopped (PID=$$targetPid)\" } else { Write-Host 'backend already stopped' }; Remove-Item '$(BACKEND_PID_FILE)' -ErrorAction SilentlyContinue; exit 0"

frontend-up:
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$targetPid = ''; if (Test-Path '$(FRONTEND_PID_FILE)') { $$targetPid = (Get-Content '$(FRONTEND_PID_FILE)' | Select-Object -First 1).Trim() }; if ($$targetPid -and (Get-Process -Id $$targetPid -ErrorAction SilentlyContinue)) { Write-Host \"frontend already running (PID=$$targetPid)\"; exit 0 }; New-Item -ItemType Directory -Path '$(LOG_DIR)' -Force | Out-Null; $$proc = Start-Process cmd -ArgumentList '/c npm run dev -- --host' -WorkingDirectory '$(CURDIR)\frontend' -RedirectStandardOutput '$(FRONTEND_OUT_LOG_FILE)' -RedirectStandardError '$(FRONTEND_ERR_LOG_FILE)' -PassThru; if (-not $$proc) { throw 'failed to start frontend process' }; $$proc.Id | Out-File '$(FRONTEND_PID_FILE)' -Encoding ascii; Write-Host \"frontend started (PID=$$($$proc.Id))\""

frontend-down:
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$targetPid = ''; if (Test-Path '$(FRONTEND_PID_FILE)') { $$targetPid = (Get-Content '$(FRONTEND_PID_FILE)' | Select-Object -First 1).Trim() }; if ($$targetPid -and (Get-Process -Id $$targetPid -ErrorAction SilentlyContinue)) { Stop-Process -Id $$targetPid -Force; Write-Host \"frontend stopped (PID=$$targetPid)\" } else { Write-Host 'frontend already stopped' }; Remove-Item '$(FRONTEND_PID_FILE)' -ErrorAction SilentlyContinue; exit 0"
