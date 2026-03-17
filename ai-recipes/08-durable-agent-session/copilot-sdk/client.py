from __future__ import annotations

import argparse
import json
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.client import OrchestrationStatus

from orchestrations import DEFAULT_MESSAGES, durable_conversation


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


def deserialize_output(serialized_output: str | None) -> list[str]:
    if not serialized_output:
        return []

    try:
        value = json.loads(serialized_output)
    except json.JSONDecodeError:
        return [serialized_output]

    if isinstance(value, list):
        return [str(item) for item in value]
    return [json.dumps(value, indent=2)]


def main() -> None:
    parser = argparse.ArgumentParser(description="Start a durable multi-turn Copilot SDK conversation.")
    parser.add_argument(
        "messages",
        nargs="*",
        default=DEFAULT_MESSAGES,
        help="Conversation prompts to send turn-by-turn.",
    )
    args = parser.parse_args()

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(durable_conversation, input=args.messages)
    print(f"Started durable conversation: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=300)
    if state is None:
        raise TimeoutError("Timed out waiting for the durable conversation to complete.")

    if state.runtime_status != OrchestrationStatus.COMPLETED:
        state.raise_if_failed()
        raise RuntimeError(f"Conversation finished with status {state.runtime_status.name}.")

    responses = deserialize_output(state.serialized_output)
    for index, message in enumerate(args.messages, start=1):
        response = responses[index - 1] if index - 1 < len(responses) else "<missing response>"
        print(f"\nTurn {index} user: {message}")
        print(f"Turn {index} assistant: {response}")


if __name__ == "__main__":
    main()
