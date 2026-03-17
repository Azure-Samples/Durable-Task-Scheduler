from __future__ import annotations

import os
import signal
import threading

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities import parse_reviews_with_copilot
from orchestrations import structured_reviews_orchestration

ENDPOINT = os.getenv("ENDPOINT", "http://localhost:8080")
TASKHUB = os.getenv("TASKHUB", "default")


def main() -> None:
    stop_event = threading.Event()

    def _handle_signal(signum: int, frame: object) -> None:
        del signum, frame
        print("\nStopping worker...")
        stop_event.set()

    signal.signal(signal.SIGINT, _handle_signal)
    signal.signal(signal.SIGTERM, _handle_signal)

    with DurableTaskSchedulerWorker(
        host_address=ENDPOINT,
        secure_channel=ENDPOINT != "http://localhost:8080",
        taskhub=TASKHUB,
        token_credential=None,
    ) as worker:
        worker.add_orchestrator(structured_reviews_orchestration)
        worker.add_activity(parse_reviews_with_copilot)
        worker.start()

        print(f"Worker listening on {ENDPOINT} (taskhub={TASKHUB}).")
        print("Press Ctrl+C to stop.")
        stop_event.wait()


if __name__ == "__main__":
    main()
