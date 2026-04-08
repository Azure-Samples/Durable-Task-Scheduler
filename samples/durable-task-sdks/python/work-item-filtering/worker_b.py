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
def add_numbers(ctx: task.ActivityContext, data: dict) -> int:
    """Activity that adds two numbers together."""
    a = data["a"]
    b = data["b"]
    logger.info(f"[Worker B] add_numbers called with a={a}, b={b}")
    return a + b


# Orchestrator function
def math_orchestrator(ctx: task.OrchestrationContext, data: dict):
    """Orchestrator that calls the add_numbers activity."""
    result = yield ctx.call_activity(add_numbers, input=data)
    return result


async def main():
    """Main entry point for Worker B — handles math orchestrations only."""
    logger.info("Starting Worker B (math worker)...")

    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"[Worker B] Using taskhub: {taskhub_name}")
    print(f"[Worker B] Using endpoint: {endpoint}")

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    ) as worker:

        # Register only the math orchestrator and activity
        worker.add_orchestrator(math_orchestrator)
        worker.add_activity(add_numbers)

        # Enable work item filtering — the worker will only receive work items
        # for the orchestrators and activities registered above
        worker.use_work_item_filters()

        # Start the worker
        worker.start()
        logger.info("[Worker B] Ready — processing only math orchestrations")

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("[Worker B] Shutdown initiated")

    logger.info("[Worker B] Stopped")

if __name__ == "__main__":
    asyncio.run(main())
