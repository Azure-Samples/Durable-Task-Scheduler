from __future__ import annotations

import argparse
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations import analysis_coordinator

LOCAL_ENDPOINT = "http://localhost:8080"
DEFAULT_QUERY = "Compare LangGraph vs Semantic Kernel vs custom orchestration for an enterprise AI assistant"


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
    parser = argparse.ArgumentParser(description="Start the recipe 07 Copilot SDK deep research orchestration.")
    parser.add_argument("query", nargs="?", default=DEFAULT_QUERY, help="Comparison question to investigate.")
    args = parser.parse_args()

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(analysis_coordinator, input=args.query)
    print(f"Started recipe 07 orchestration: {instance_id}")
    state = client.wait_for_orchestration_completion(instance_id, timeout=300)

    if state is None:
        raise TimeoutError("Timed out waiting for the recipe 07 orchestration to complete.")

    print("\nDeep research report:\n")
    print(state.serialized_output)


if __name__ == "__main__":
    main()
