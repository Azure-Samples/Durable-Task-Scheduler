import argparse
import json
import logging
import os

from azure.identity import DefaultAzureCredential
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker
from durabletask.client import OrchestrationStatus

from workflows import call_partner, process_order, record_order, validate_order


def run_order(client: DurableTaskSchedulerClient, order_id: str) -> str:
    if not order_id.strip():
        raise ValueError("Order ID must not be empty.")
    instance_id = client.schedule_new_orchestration(
        process_order, input=order_id, tags={"DTS_TaskQueue": "orders"}
    )
    print(f"Started {instance_id} with DTS_TaskQueue=orders")
    state = client.wait_for_orchestration_completion(instance_id, timeout=120)
    if state is None:
        raise TimeoutError(f"Instance {instance_id} did not complete within 120 seconds.")
    if state.runtime_status != OrchestrationStatus.COMPLETED:
        raise RuntimeError(
            f"Instance {instance_id}: {state.runtime_status}. {state.failure_details}"
        )
    output = json.loads(state.serialized_output)
    expected = f"Recorded {order_id}; Partner accepted {order_id}; Validated {order_id}"
    if output != expected:
        raise RuntimeError(f"Unexpected output: {output!r}")
    return output


def main():
    parser = argparse.ArgumentParser(description="Route order work to DTS task queues.")
    parser.add_argument("order_id", nargs="?", default="order-001")
    args = parser.parse_args()
    logging.basicConfig(level=logging.INFO)
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()
    config = {
        "host_address": endpoint,
        "taskhub": os.getenv("TASKHUB", "default"),
        "secure_channel": endpoint != "http://localhost:8080",
        "token_credential": credential,
    }
    try:
        with DurableTaskSchedulerWorker(**config) as worker:
            worker.add_orchestrator(process_order)
            worker.add_orchestrator(validate_order)
            worker.add_activity(record_order)
            worker.add_activity(call_partner)
            # Do not opt in to work-item filtering for named task queues.
            worker.start()
            with DurableTaskSchedulerClient(**config) as client:
                print(run_order(client, args.order_id))
    finally:
        if credential is not None:
            credential.close()


if __name__ == "__main__":
    main()
