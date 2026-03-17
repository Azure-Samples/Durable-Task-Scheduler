from __future__ import annotations

import argparse
import json
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

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
    parser = argparse.ArgumentParser(description="Start the multi-agent coordinator orchestration.")
    parser.add_argument("task", help="Complex task description for the coordinator.")
    args = parser.parse_args()

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(coordinator, input=args.task)
    print(f"Started multi-agent orchestration: {instance_id}")
    state = client.wait_for_orchestration_completion(instance_id, timeout=300)

    if state is None:
        raise TimeoutError("Timed out waiting for the multi-agent orchestration to complete.")

    print("\nFinal result:\n")
    print(json.dumps(json.loads(state.serialized_output), indent=2))


if __name__ == "__main__":
    main()
