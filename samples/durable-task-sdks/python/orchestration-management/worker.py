import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# Activity functions
def process_batch(ctx: task.ActivityContext, data: dict) -> dict:
    """Activity that simulates processing a batch of data."""
    batch_id = data["batch_id"]
    item_count = data["item_count"]
    logger.info(f"Processing batch {batch_id} with {item_count} items")
    return {
        "batch_id": batch_id,
        "items_processed": item_count,
        "status": "success",
    }


# Orchestrator function
def data_processing_orchestrator(ctx: task.OrchestrationContext, data: dict):
    """Orchestrator that processes a batch of data."""
    result = yield ctx.call_activity(process_batch, input=data)
    return result


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Orchestration Management pattern worker...")

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

        # Register activities and orchestrator
        worker.add_activity(process_batch)
        worker.add_orchestrator(data_processing_orchestrator)

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
