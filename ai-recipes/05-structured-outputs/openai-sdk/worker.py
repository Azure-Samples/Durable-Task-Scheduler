from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.invoke_model import invoke_model
from orchestrations.clean_data import clean_data


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
        worker.add_orchestrator(clean_data)
        worker.add_activity(invoke_model)
        worker.start()

        print("Structured outputs worker is running. Press Ctrl+C to stop.")
        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print("Stopping worker...")


if __name__ == "__main__":
    main()
