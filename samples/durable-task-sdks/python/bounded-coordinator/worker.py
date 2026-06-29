import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Tunables: how many batches the source yields, and how many items per batch.
TOTAL_BATCHES = 3
ITEMS_PER_BATCH = 5
MAX_ITEMS = 50


# Activity that returns a bounded batch of work items
def get_next_batch(ctx: task.ActivityContext, input: dict) -> dict:
    """Return a bounded batch of work items for the given cursor.

    The batch number is derived deterministically from the cursor so the
    activity is stateless and safe across retries / scale-out.
    """
    cursor = input.get("cursor")
    max_items = input.get("max_items", MAX_ITEMS)

    batch_number = 1 if cursor is None else int(cursor.split("-")[-1]) + 1

    if batch_number > TOTAL_BATCHES:
        return {"items": [], "next_cursor": None, "has_more": False}

    count = min(ITEMS_PER_BATCH, max_items)
    items = [
        {
            "id": f"item-{batch_number}-{i}",
            "tenant_id": f"tenant-{i}",
            "payload": f"data-{batch_number}-{i}",
        }
        for i in range(1, count + 1)
    ]

    return {
        "items": items,
        "next_cursor": f"cursor-{batch_number}",
        "has_more": batch_number < TOTAL_BATCHES,
    }


# Activity that applies a change for a single work item
def apply_change(ctx: task.ActivityContext, item: dict) -> str:
    """Apply the change for a single work item."""
    logger.info(f"Applying change for item {item['id']}, tenant {item['tenant_id']}")
    return f"processed:{item['id']}"


# Short-lived child orchestration that processes a single work item
def process_item_orchestrator(ctx: task.OrchestrationContext, item: dict) -> str:
    """Child orchestration that processes a single work item.

    Completes quickly and does not accumulate history.
    """
    olog = ctx.create_replay_safe_logger(logger)
    olog.info(f"Processing item {item['id']} for tenant {item['tenant_id']}")

    result = yield ctx.call_activity(apply_change, input=item)
    return result


# Bounded coordinator orchestration
def coordinator_orchestrator(ctx: task.OrchestrationContext, state: dict | None):
    """Bounded coordinator that fans out child work, waits for all children,
    then resets via continue_as_new.

    This prevents unbounded history growth that can occur with long-lived
    message-pump / coordinator orchestrations.
    """
    olog = ctx.create_replay_safe_logger(logger)

    # Carry-forward state: a cursor into the work source and a batch counter.
    state = state or {"cursor": None, "batch_number": 0}
    cursor = state["cursor"]
    batch_number = state["batch_number"]

    olog.info(f"Coordinator batch {batch_number}, cursor={cursor or '(start)'}")

    # Step 1: Read a BOUNDED batch of work items.
    batch = yield ctx.call_activity(
        get_next_batch, input={"cursor": cursor, "max_items": MAX_ITEMS}
    )

    if not batch["items"]:
        olog.info("No more items to process. Coordinator completing.")
        return {"total_batches": batch_number, "completed": True}

    # Step 2: Fan out child sub-orchestrations for this batch.
    olog.info(f"Processing batch of {len(batch['items'])} items")
    child_tasks = [
        ctx.call_sub_orchestrator(process_item_orchestrator, input=item)
        for item in batch["items"]
    ]

    # Step 3: Wait for ALL children to complete before resetting.
    # This is critical - never call continue_as_new while children are still running.
    yield task.when_all(child_tasks)

    olog.info(f"Batch {batch_number + 1} complete. {len(batch['items'])} items processed.")

    # Step 4: continue_as_new with compact carry-forward state. This resets the
    # orchestration history, preventing unbounded growth.
    if batch["has_more"]:
        ctx.continue_as_new({
            "cursor": batch["next_cursor"],
            "batch_number": batch_number + 1,
        })
        return  # unreachable after continue_as_new

    return {"total_batches": batch_number + 1, "completed": True}


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Bounded Coordinator pattern worker...")

    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    ) as worker:

        # Register activities and orchestrators
        worker.add_activity(get_next_batch)
        worker.add_activity(apply_change)
        worker.add_orchestrator(process_item_orchestrator)
        worker.add_orchestrator(coordinator_orchestrator)

        # Start the worker
        worker.start()

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("Worker shutdown initiated")

    logger.info("Worker stopped")

if __name__ == "__main__":
    asyncio.run(main())
