import asyncio
import logging
import os
import time
from datetime import datetime, timedelta, timezone

from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(message)s",
    datefmt="%H:%M:%S",
)
logger = logging.getLogger("Client")

# Schedule a batch of orchestrations on a fixed interval so you can watch the
# workers scale over time.
ORCHESTRATIONS_PER_BATCH = 3
INTERVAL_SECONDS = 30
TOTAL_DURATION_MINUTES = 10


async def main():
    logger.info("=== Work Item Filtering Demo — Client ===")

    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    managed_identity_client_id = os.getenv("AZURE_MANAGED_IDENTITY_CLIENT_ID")

    print(f"[Client] Using taskhub: {taskhub_name}")
    print(f"[Client] Using endpoint: {endpoint}")

    # Use no credential for the local emulator, a user-assigned managed identity
    # in Container Apps, or DefaultAzureCredential for local Azure development.
    if endpoint == "http://localhost:8080":
        credential = None
    elif managed_identity_client_id:
        credential = DefaultAzureCredential(managed_identity_client_id=managed_identity_client_id)
    else:
        credential = DefaultAzureCredential()

    client = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    )

    interval = timedelta(seconds=INTERVAL_SECONDS)
    deadline = datetime.now(timezone.utc) + timedelta(minutes=TOTAL_DURATION_MINUTES)

    total_completed = 0
    total_failed = 0
    batch_number = 0

    logger.info(
        "Will schedule %d orchestrations every %ds for %d minutes.",
        ORCHESTRATIONS_PER_BATCH,
        INTERVAL_SECONDS,
        TOTAL_DURATION_MINUTES,
    )
    logger.info("(Make sure the Orchestrator, Validator, and Shipper workers are all running)\n")

    while datetime.now(timezone.utc) < deadline:
        batch_number += 1
        logger.info("--- Batch #%d at %s ---", batch_number, datetime.now().strftime("%H:%M:%S"))

        instance_ids = []
        for i in range(1, ORCHESTRATIONS_PER_BATCH + 1):
            order_id = f"ORD-B{batch_number:03d}-{i:03d}"
            logger.info("Scheduling orchestration with orderId='%s'...", order_id)
            instance_id = client.schedule_new_orchestration(
                "order_processing_orchestrator", input=order_id
            )
            instance_ids.append(instance_id)
            logger.info("  -> Scheduled with InstanceId=%s", instance_id)

        # Wait for all orchestrations in this batch to complete.
        batch_completed = 0
        batch_failed = 0
        for instance_id in instance_ids:
            try:
                state = client.wait_for_orchestration_completion(instance_id, timeout=120)
                if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
                    batch_completed += 1
                    logger.info(
                        "COMPLETED | InstanceId=%s | Output: %s",
                        instance_id,
                        state.serialized_output,
                    )
                elif state:
                    batch_failed += 1
                    logger.error(
                        "FAILED    | InstanceId=%s | Status=%s | Error: %s",
                        instance_id,
                        state.runtime_status,
                        state.failure_details,
                    )
            except Exception as ex:  # noqa: BLE001
                batch_failed += 1
                logger.error("Error waiting for orchestration %s: %s", instance_id, ex)

        total_completed += batch_completed
        total_failed += batch_failed
        logger.info(
            "Batch #%d results: %d completed, %d failed",
            batch_number,
            batch_completed,
            batch_failed,
        )

        # Wait for the next interval (unless we've passed the deadline).
        now = datetime.now(timezone.utc)
        if now < deadline:
            remaining = deadline - now
            wait_time = min(remaining, interval)
            logger.info(
                "Next batch in %.0fs (deadline in %.1f min)\n",
                wait_time.total_seconds(),
                remaining.total_seconds() / 60,
            )
            await asyncio.sleep(wait_time.total_seconds())

    logger.info(
        "\n=== FINAL RESULTS: %d completed, %d failed across %d batches ===",
        total_completed,
        total_failed,
        batch_number,
    )

    # Keep the process alive so Container Apps doesn't mark it as failed.
    logger.info("Demo complete. Staying alive — press Ctrl+C to exit.")
    while True:
        time.sleep(3600)


if __name__ == "__main__":
    asyncio.run(main())
