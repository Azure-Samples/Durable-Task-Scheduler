import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# Activity function
def send_report(ctx: task.ActivityContext, region: str) -> str:
    """Activity that simulates generating and sending a periodic report."""
    logger.info(f"Generating report for region '{region}'")
    return f"Report for '{region}' generated"


# Target orchestration that the schedule starts on each run
def report_orchestrator(ctx: task.OrchestrationContext, region: str):
    """Orchestration started on every schedule tick."""
    result = yield ctx.call_activity(send_report, input=region)
    return result


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Scheduled Tasks pattern worker...")

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

        # Register the target activity and orchestrator
        worker.add_activity(send_report)
        worker.add_orchestrator(report_orchestrator)

        # Register the schedule entity and operation orchestrator that power
        # the recurring schedule feature. This must be called on every worker
        # that participates in scheduled tasks.
        worker.configure_scheduled_tasks()

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
