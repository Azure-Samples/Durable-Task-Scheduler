from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.llm_activity import llm_activity
from activities.search import search_comparison
from activities.synthesize import create_comparison_report
from orchestrations.analysis_coordinator import analysis_coordinator
from orchestrations.dimension_analyst import dimension_analyst


LOCAL_ENDPOINT = "http://localhost:8080"


def get_connection_config() -> dict:
    endpoint = os.getenv("ENDPOINT", LOCAL_ENDPOINT)
    taskhub = os.getenv("TASKHUB", "default")
    is_local = endpoint == LOCAL_ENDPOINT

    credential = None
    if not is_local:
        from azure.identity import DefaultAzureCredential

        credential = DefaultAzureCredential()

    return {
        "host_address": endpoint,
        "taskhub": taskhub,
        "secure_channel": not is_local,
        "token_credential": credential,
    }


def main() -> None:
    with DurableTaskSchedulerWorker(**get_connection_config()) as worker:
        worker.add_orchestrator(analysis_coordinator)
        worker.add_orchestrator(dimension_analyst)
        worker.add_activity(llm_activity)
        worker.add_activity(search_comparison)
        worker.add_activity(create_comparison_report)
        worker.start()
        print("Competitive analysis worker is running. Press Ctrl+C to stop.")

        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print("Stopping worker...")


if __name__ == "__main__":
    main()
