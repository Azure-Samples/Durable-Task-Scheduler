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
def say_hello(ctx: task.ActivityContext, name: str) -> str:
    """Activity that returns a greeting."""
    logger.info(f"[Worker A] say_hello called with name: {name}")
    return f"Hello, {name}!"


# Orchestrator function
def greeting_orchestrator(ctx: task.OrchestrationContext, name: str):
    """Orchestrator that calls the say_hello activity."""
    result = yield ctx.call_activity(say_hello, input=name)
    return result


async def main():
    """Main entry point for Worker A — handles greeting orchestrations only."""
    logger.info("Starting Worker A (greeting worker)...")

    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"[Worker A] Using taskhub: {taskhub_name}")
    print(f"[Worker A] Using endpoint: {endpoint}")

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    ) as worker:

        # Register only the greeting orchestrator and activity
        worker.add_orchestrator(greeting_orchestrator)
        worker.add_activity(say_hello)

        # Enable work item filtering — the worker will only receive work items
        # for the orchestrators and activities registered above
        worker.use_work_item_filters()

        # Start the worker
        worker.start()
        logger.info("[Worker A] Ready — processing only greeting orchestrations")

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("[Worker A] Shutdown initiated")

    logger.info("[Worker A] Stopped")

if __name__ == "__main__":
    asyncio.run(main())
