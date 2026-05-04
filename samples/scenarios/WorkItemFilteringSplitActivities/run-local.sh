#!/usr/bin/env bash
# Runs the WorkItemFilteringSplitActivities sample locally.
# - Starts the DTS emulator (if not already running)
# - Builds the solution
# - Launches the three workers and the client, each in its own log file
# - Press Ctrl+C to stop everything

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

LOG_DIR="$SCRIPT_DIR/.logs"
PID_FILE="$SCRIPT_DIR/.logs/pids"
EMULATOR_NAME="dts-emulator"
EMULATOR_IMAGE="mcr.microsoft.com/durable-task/emulator:latest"

mkdir -p "$LOG_DIR"
: > "$PID_FILE"

cleanup() {
    echo ""
    echo "[run-local] Stopping workers and client..."
    if [[ -f "$PID_FILE" ]]; then
        while read -r pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null || true
            fi
        done < "$PID_FILE"
    fi
    if [[ "${KEEP_EMULATOR:-0}" != "1" ]]; then
        echo "[run-local] Stopping DTS emulator container ($EMULATOR_NAME)..."
        docker rm -f "$EMULATOR_NAME" >/dev/null 2>&1 || true
    else
        echo "[run-local] Leaving DTS emulator running (KEEP_EMULATOR=1)."
    fi
    echo "[run-local] Done."
}
trap cleanup EXIT INT TERM

# 1. Ensure Docker is available
if ! command -v docker >/dev/null 2>&1; then
    echo "[run-local] ERROR: Docker is required but not found in PATH." >&2
    exit 1
fi

# 2. Start the DTS emulator if needed
if docker ps --format '{{.Names}}' | grep -q "^${EMULATOR_NAME}$"; then
    echo "[run-local] DTS emulator already running."
else
    if docker ps -a --format '{{.Names}}' | grep -q "^${EMULATOR_NAME}$"; then
        echo "[run-local] Removing stale emulator container..."
        docker rm -f "$EMULATOR_NAME" >/dev/null
    fi
    echo "[run-local] Pulling DTS emulator image..."
    docker pull "$EMULATOR_IMAGE" >/dev/null
    echo "[run-local] Starting DTS emulator (dashboard: http://localhost:8082)..."
    docker run -d --name "$EMULATOR_NAME" -p 8080:8080 -p 8082:8082 "$EMULATOR_IMAGE" >/dev/null
fi

# 3. Build the solution once so each `dotnet run` starts fast
echo "[run-local] Building solution..."
dotnet build WorkItemFilteringSplitActivities.sln --nologo -v minimal

# 4. Launch workers and client
start_proc() {
    local name="$1"
    local project="$2"
    local log_file="$LOG_DIR/${name}.log"
    echo "[run-local] Starting $name (logs: $log_file)"
    # --no-build: solution already built above
    dotnet run --no-build --project "$project" >"$log_file" 2>&1 &
    echo $! >> "$PID_FILE"
}

start_proc "orchestrator-worker" "src/OrchestratorWorker"
start_proc "validator-worker"    "src/ValidatorWorker"
start_proc "shipper-worker"      "src/ShipperWorker"

# Give workers a moment to connect before the client starts scheduling
sleep 3

start_proc "client" "src/Client"

echo ""
echo "[run-local] All processes started. Tailing logs (Ctrl+C to stop everything)..."
echo "[run-local] Logs are also saved under $LOG_DIR/"
echo ""

# 5. Tail all logs until the user interrupts
tail -n +1 -F \
    "$LOG_DIR/orchestrator-worker.log" \
    "$LOG_DIR/validator-worker.log" \
    "$LOG_DIR/shipper-worker.log" \
    "$LOG_DIR/client.log"
