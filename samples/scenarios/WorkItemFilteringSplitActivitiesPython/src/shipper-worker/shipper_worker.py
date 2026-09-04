import asyncio
import logging
import os
import random

from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(message)s",
    datefmt="%H:%M:%S",
)
logger = logging.getLogger("Shipper")

WORKER_NAME = "Shipper Worker"


def ship_order(ctx: task.ActivityContext, order_id: str) -> str:
    """Ships an order. Registered ONLY in the Shipper Worker,
    so DTS routes ship_order work items exclusively to this worker.
    """
    logger.info(
        "[Shipper] Activity | Name=ship_order | InstanceId=%s | Shipping order '%s'...",
        ctx.orchestration_id,
        order_id,
    )

    tracking_number = f"TRACK-{order_id}-{random.randint(1000, 9999)}"
    result = f"Shipped with tracking {tracking_number}"

    logger.info(
        "[Shipper] Activity | Name=ship_order | InstanceId=%s | Result: %s",
        ctx.orchestration_id,
        result,
    )
    return result


async def main():
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    managed_identity_client_id = os.getenv("AZURE_MANAGED_IDENTITY_CLIENT_ID")

    print(f"[{WORKER_NAME}] Using taskhub: {taskhub_name}")
    print(f"[{WORKER_NAME}] Using endpoint: {endpoint}")

    # Use no credential for the local emulator, a user-assigned managed identity
    # in Container Apps, or DefaultAzureCredential for local Azure development.
    if endpoint == "http://localhost:8080":
        credential = None
    elif managed_identity_client_id:
        credential = DefaultAzureCredential(managed_identity_client_id=managed_identity_client_id)
    else:
        credential = DefaultAzureCredential()

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    ) as worker:

        # Register ONLY the ship_order activity — no orchestrations.
        worker.add_activity(ship_order)

        # Auto-generate work item filters from the registry, so this worker
        # receives ONLY ship_order activity work items.
        worker.use_work_item_filters()

        worker.start()
        logger.info("[%s] Ready — processing ship_order activity only.", WORKER_NAME)

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("[%s] Shutdown initiated", WORKER_NAME)

    logger.info("[%s] Stopped", WORKER_NAME)


if __name__ == "__main__":
    asyncio.run(main())
