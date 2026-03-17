from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities import run_executor_agent, run_planner_agent, run_reviewer_agent
from orchestrations import SharedStateEntity, executor_session, multi_agent_orchestration

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
        worker.add_orchestrator(multi_agent_orchestration)
        worker.add_orchestrator(executor_session)
        worker.add_entity(SharedStateEntity)
        worker.add_activity(run_planner_agent)
        worker.add_activity(run_executor_agent)
        worker.add_activity(run_reviewer_agent)
        worker.start()
        print("Recipe 07 Copilot SDK worker is running. Press Ctrl+C to stop.")

        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print("Stopping worker...")


if __name__ == "__main__":
    main()
