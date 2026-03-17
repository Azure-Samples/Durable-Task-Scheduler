from __future__ import annotations

import argparse
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations import multi_agent_orchestration

LOCAL_ENDPOINT = "http://localhost:8080"
DEFAULT_TASK = "Design and validate a rollout plan for a durable Copilot-powered support assistant"
DEFAULT_WAIT_TIMEOUT_SECONDS = int(os.getenv("WAIT_TIMEOUT_SECONDS", "600"))


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
    parser = argparse.ArgumentParser(description="Start the recipe 08 Copilot SDK multi-agent orchestration.")
    parser.add_argument("task_description", nargs="?", default=DEFAULT_TASK, help="Task for the planner/executor/reviewer flow.")
    args = parser.parse_args()

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(multi_agent_orchestration, input=args.task_description)
    print(f"Started recipe 08 orchestration: {instance_id}")
    state = client.wait_for_orchestration_completion(instance_id, timeout=DEFAULT_WAIT_TIMEOUT_SECONDS)

    if state is None:
        raise TimeoutError(
            f"Timed out waiting {DEFAULT_WAIT_TIMEOUT_SECONDS}s for the recipe 08 orchestration to complete."
        )

    print("\nMulti-agent review:\n")
    print(state.serialized_output)


if __name__ == "__main__":
    main()
