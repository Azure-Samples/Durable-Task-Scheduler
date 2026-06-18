"""Remote worker image entrypoint for the On-demand Sandboxes code-interpreter demo.

Runs the untrusted, LLM-generated Python in an isolated DTS-managed sandbox.
Each invocation writes the CSV partition and generated script to /tmp and shells
out to python3, capturing stdout/stderr and the exit code.
"""

import os
import subprocess
import threading
import uuid

from durabletask import task
from durabletask.azuremanaged.preview.sandboxes import SandboxWorker

from activities import EXECUTE_CODE

EXECUTION_TIMEOUT_SECONDS = 30
MAX_DISPLAY_LINES = 30


def execute_code(ctx: task.ActivityContext, payload: dict) -> dict:
    """Activity that runs the generated pandas script inside the sandbox container."""
    sandbox_name = os.getenv("DTS_SANDBOX_ID") or os.uname().nodename
    python_code = payload["code"]
    csv_data = payload["csv"]

    print(f"[sandbox] Starting execute_code in sandbox '{sandbox_name}' (pid={os.getpid()})")
    print("[sandbox] This is isolated on-demand sandbox compute managed by DTS.")

    code_lines = python_code.split("\n")
    byte_count = len(python_code.encode("utf-8"))
    print(f"[sandbox] Received generated Python ({len(code_lines)} lines, {byte_count} bytes)")
    print("[sandbox] --- generated script ---")
    for line in code_lines[:MAX_DISPLAY_LINES]:
        print(line)
    if len(code_lines) > MAX_DISPLAY_LINES:
        print(f"... (truncated, {len(code_lines) - MAX_DISPLAY_LINES} more lines)")
    print("[sandbox] --- end script ---")

    work_dir = os.path.join("/tmp", f"run-{uuid.uuid4().hex}")
    os.makedirs(work_dir, exist_ok=True)
    print(f"[sandbox] Created isolated work directory: {work_dir}")

    csv_path = os.path.join(work_dir, "data.csv")
    script_path = os.path.join(work_dir, "script.py")
    with open(csv_path, "w", encoding="utf-8") as handle:
        handle.write(csv_data)
    with open(script_path, "w", encoding="utf-8") as handle:
        handle.write(python_code)
    print(f"[sandbox] Wrote dataset: {csv_path}")
    print(f"[sandbox] Wrote generated script: {script_path}")

    # The generated script reads /tmp/data.csv. Copy into the canonical location
    # so the LLM doesn't need to know about per-invocation working directories.
    with open("/tmp/data.csv", "w", encoding="utf-8") as handle:
        handle.write(csv_data)
    print("[sandbox] Mounted dataset at expected path: /tmp/data.csv")

    csv_lines = [line for line in csv_data.split("\n") if line.strip()]
    if csv_lines:
        csv_headers = csv_lines[0].split(",")
        print(f"[sandbox] Dataset loaded: {len(csv_lines) - 1} rows x "
              f"{len(csv_headers)} columns [{', '.join(csv_headers)}]")

    print(f"[sandbox] Executing command: python3 {script_path}")
    try:
        completed = subprocess.run(
            ["python3", script_path],
            cwd=work_dir,
            capture_output=True,
            text=True,
            timeout=EXECUTION_TIMEOUT_SECONDS,
            check=False)
    except subprocess.TimeoutExpired:
        print(f"[sandbox] ERROR: Timeout: execution exceeded {EXECUTION_TIMEOUT_SECONDS} seconds.")
        return {
            "stdout": "",
            "stderr": f"Execution timed out after {EXECUTION_TIMEOUT_SECONDS} seconds.",
            "exit_code": 124,
        }

    stdout = completed.stdout or ""
    stderr = completed.stderr or ""
    print(f"[sandbox] Python process completed (exit code {completed.returncode})")
    if stdout.strip():
        print("[sandbox] stdout from generated script:")
        print(stdout.rstrip())
    else:
        print("[sandbox] stdout from generated script: <empty>")
    if completed.returncode != 0 and stderr.strip():
        print(f"[sandbox] ERROR: {stderr.rstrip()}")
    if completed.returncode == 0:
        print("[sandbox] Returning captured stdout to the orchestrator.")

    return {"stdout": stdout, "stderr": stderr, "exit_code": completed.returncode}


execute_code.__name__ = EXECUTE_CODE.name


def main() -> None:
    with SandboxWorker() as worker:
        worker.add_activity(execute_code, version=EXECUTE_CODE.version)
        worker.start()
        print("Python on-demand sandbox worker is running. Press Ctrl+C to stop.")
        try:
            threading.Event().wait()
        except KeyboardInterrupt:
            # Expected on Ctrl+C: let the context manager stop the worker gracefully.
            pass


if __name__ == "__main__":
    main()
