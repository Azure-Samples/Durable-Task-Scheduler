#!/usr/bin/env bash
# Runs the WorkItemFilteringSplitActivitiesPython sample locally.
# - Starts the DTS emulator (if not already running)
# - Creates a shared virtual environment and installs dependencies
# - Launches the three workers and the client, each in its own log file
# - Press Ctrl+C to stop everything

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

LOG_DIR="$SCRIPT_DIR/.logs"
PID_FILE="$SCRIPT_DIR/.logs/pids"
VENV_DIR="$SCRIPT_DIR/.venv"
EMULATOR_NAME="dts-emulator"
EMULATOR_IMAGE="mcr.microsoft.com/dts/dts-emulator:latest"

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

# 3. Create a shared virtual environment and install dependencies
if [[ ! -d "$VENV_DIR" ]]; then
    echo "[run-local] Creating virtual environment at $VENV_DIR..."
    python3 -m venv "$VENV_DIR"
fi
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
echo "[run-local] Installing dependencies..."
pip install --quiet --upgrade pip
pip install --quiet -r src/client/requirements.txt

# 4. Launch workers and client
start_proc() {
    local name="$1"
    local script="$2"
    local log_file="$LOG_DIR/${name}.log"
    echo "[run-local] Starting $name (logs: $log_file)"
    python "$script" >"$log_file" 2>&1 &
    echo $! >> "$PID_FILE"
}

start_proc "orchestrator-worker" "src/orchestrator-worker/orchestrator_worker.py"
start_proc "validator-worker"    "src/validator-worker/validator_worker.py"
start_proc "shipper-worker"      "src/shipper-worker/shipper_worker.py"

# Give workers a moment to connect before the client starts scheduling
sleep 3

start_proc "client" "src/client/client.py"

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
