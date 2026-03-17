"""Worker process that hosts the durable GitHub inspector orchestrations."""

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.github_api import fetch_github_api
from orchestrations.github_workflows import GetRecentActivity, GetRepoInfo

DEFAULT_ENDPOINT = "http://localhost:8080"
DEFAULT_TASKHUB = "default"


def get_connection_settings() -> dict:
    """Return the DTS emulator connection settings."""
    endpoint = os.getenv("DTS_ENDPOINT", DEFAULT_ENDPOINT)
    return {
        "host_address": endpoint,
        "taskhub": os.getenv("DTS_TASKHUB", DEFAULT_TASKHUB),
        "token_credential": None,
        "secure_channel": endpoint.startswith("https://"),
    }


def main() -> None:
    """Register orchestrations/activities and start polling the scheduler."""
    with DurableTaskSchedulerWorker(**get_connection_settings()) as worker:
        worker.add_orchestrator(GetRepoInfo)
        worker.add_orchestrator(GetRecentActivity)
        worker.add_activity(fetch_github_api)
        worker.start()

        print("GitHub inspector worker is running. Press Ctrl+C to stop.")
        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print("Stopping GitHub inspector worker.")


if __name__ == "__main__":
    main()
