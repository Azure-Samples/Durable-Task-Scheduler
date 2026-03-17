from __future__ import annotations

import json
import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.client import OrchestrationStatus

from orchestrations import DEFAULT_REVIEW_DATA, structured_reviews_orchestration

ENDPOINT = os.getenv("ENDPOINT", "http://localhost:8080")
TASKHUB = os.getenv("TASKHUB", "default")


def main() -> None:
    raw_review_data = ' '.join(sys.argv[1:]) if len(sys.argv) > 1 else DEFAULT_REVIEW_DATA

    client = DurableTaskSchedulerClient(
        host_address=ENDPOINT,
        secure_channel=ENDPOINT != "http://localhost:8080",
        taskhub=TASKHUB,
        token_credential=None,
    )

    instance_id = client.schedule_new_orchestration(structured_reviews_orchestration, input=raw_review_data)
    print(f"Started orchestration: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=120)
    if state is None:
        raise TimeoutError("Timed out waiting for the orchestration to complete.")

    if state.runtime_status != OrchestrationStatus.COMPLETED:
        state.raise_if_failed()
        raise RuntimeError(f"Orchestration finished with status {state.runtime_status.name}.")

    payload = json.loads(state.serialized_output or '{}')
    print(json.dumps(payload, indent=2))


if __name__ == "__main__":
    main()
