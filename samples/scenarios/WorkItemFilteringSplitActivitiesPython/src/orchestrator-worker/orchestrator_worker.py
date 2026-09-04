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
logger = logging.getLogger("Orchestrator")

WORKER_NAME = "Orchestrator Worker"


def order_processing_orchestrator(ctx: task.OrchestrationContext, order_id: str):
    """Calls two activities sequentially:

    1. validate_order  (routed to the Validator Worker)
    2. ship_order      (routed to the Shipper Worker)

    Because each activity is registered in a different worker process, DTS routes
    each activity work item to the correct worker via work item filtering. This
    worker registers ONLY the orchestrator, so it never receives activity work items.
    """
    if not ctx.is_replaying:
        logger.info(
            "[Orchestrator] Orchestration | Name=order_processing_orchestrator | "
            "InstanceId=%s | Processing order '%s'",
            ctx.instance_id,
            order_id,
        )

    # Step 1: Validate the order (routed to the Validator Worker).
    if not ctx.is_replaying:
        logger.info(
            "[Orchestrator] Orchestration | InstanceId=%s | Dispatching validate_order to Validator Worker...",
            ctx.instance_id,
        )
    validation_result = yield ctx.call_activity("validate_order", input=order_id)

    # Step 2: Ship the order (routed to the Shipper Worker).
    if not ctx.is_replaying:
        logger.info(
            "[Orchestrator] Orchestration | InstanceId=%s | Dispatching ship_order to Shipper Worker...",
            ctx.instance_id,
        )
    shipping_result = yield ctx.call_activity("ship_order", input=order_id)

    combined = (
        f"Order '{order_id}' => Validation: [{validation_result}], "
        f"Shipping: [{shipping_result}]"
    )

    if not ctx.is_replaying:
        logger.info(
            "[Orchestrator] Orchestration | InstanceId=%s | Completed: %s",
            ctx.instance_id,
            combined,
        )

    return combined


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

        # Register ONLY the orchestrator — no activities.
        worker.add_orchestrator(order_processing_orchestrator)

        # Auto-generate work item filters from the registry, so this worker
        # receives ONLY orchestration work items — never activity work items.
        worker.use_work_item_filters()

        worker.start()
        logger.info("[%s] Ready — processing orchestrations only.", WORKER_NAME)

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("[%s] Shutdown initiated", WORKER_NAME)

    logger.info("[%s] Stopped", WORKER_NAME)


if __name__ == "__main__":
    asyncio.run(main())
