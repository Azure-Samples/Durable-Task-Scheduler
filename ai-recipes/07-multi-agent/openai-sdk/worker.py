from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.llm_activity import llm_activity
from activities.tool_invoker import tool_invoker
from entities.shared_state import sharedstate
from orchestrations.agent import agent
from orchestrations.coordinator import coordinator


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
        worker.add_orchestrator(coordinator)
        worker.add_orchestrator(agent)
        worker.add_activity(llm_activity)
        worker.add_activity(tool_invoker)
        worker.add_entity(sharedstate)
        worker.start()
        print("Multi-agent worker is running. Press Ctrl+C to stop.")

        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print("Stopping worker...")


if __name__ == "__main__":
    main()
