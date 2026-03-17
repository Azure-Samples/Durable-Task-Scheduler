from __future__ import annotations

import json
import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.client import OrchestrationStatus

from orchestrations import DEFAULT_INPUT, agent_orchestration

ENDPOINT = os.getenv("ENDPOINT", "http://localhost:8080")
TASKHUB = os.getenv("TASKHUB", "default")


def _deserialize_output(serialized_output: str | None) -> str:
    if not serialized_output:
        return ""

    try:
        value = json.loads(serialized_output)
    except json.JSONDecodeError:
        return serialized_output

    return value if isinstance(value, str) else json.dumps(value, indent=2)


def main() -> None:
    user_input = ' '.join(sys.argv[1:]) if len(sys.argv) > 1 else DEFAULT_INPUT

    client = DurableTaskSchedulerClient(
        host_address=ENDPOINT,
        secure_channel=ENDPOINT != "http://localhost:8080",
        taskhub=TASKHUB,
        token_credential=None,
    )

    instance_id = client.schedule_new_orchestration(agent_orchestration, input=user_input)
    print(f"Started orchestration: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=120)
    if state is None:
        raise TimeoutError("Timed out waiting for the orchestration to complete.")

    if state.runtime_status != OrchestrationStatus.COMPLETED:
        state.raise_if_failed()
        raise RuntimeError(f"Orchestration finished with status {state.runtime_status.name}.")

    print(_deserialize_output(state.serialized_output))


if __name__ == "__main__":
    main()
