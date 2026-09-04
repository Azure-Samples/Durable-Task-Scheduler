import asyncio
import logging
import os

from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(message)s",
    datefmt="%H:%M:%S",
)
logger = logging.getLogger("Validator")

WORKER_NAME = "Validator Worker"


def validate_order(ctx: task.ActivityContext, order_id: str) -> str:
    """Validates an incoming order. Registered ONLY in the Validator Worker,
    so DTS routes validate_order work items exclusively to this worker.
    """
    logger.info(
        "[Validator] Activity | Name=validate_order | InstanceId=%s | Validating order '%s'...",
        ctx.orchestration_id,
        order_id,
    )

    result = f"Order {order_id} is valid"

    logger.info(
        "[Validator] Activity | Name=validate_order | InstanceId=%s | Result: %s",
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

        # Register ONLY the validate_order activity — no orchestrations.
        worker.add_activity(validate_order)

        # Auto-generate work item filters from the registry, so this worker
        # receives ONLY validate_order activity work items.
        worker.use_work_item_filters()

        worker.start()
        logger.info("[%s] Ready — processing validate_order activity only.", WORKER_NAME)

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("[%s] Shutdown initiated", WORKER_NAME)

    logger.info("[%s] Stopped", WORKER_NAME)


if __name__ == "__main__":
    asyncio.run(main())
