import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main():
    """Main entry point that starts the bounded coordinator orchestration."""
    logger.info("Starting Bounded Coordinator client...")

    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    client = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    )

    # Start the coordinator with no input; it initializes its own carry-forward state.
    instance_id = client.schedule_new_orchestration("coordinator_orchestrator")
    logger.info(f"Started coordinator_orchestrator, instance ID: {instance_id}")

    # Wait for the coordinator to finish processing all batches. Each batch
    # resets history via continue_as_new, but the instance ID stays the same.
    state = client.wait_for_orchestration_completion(instance_id, timeout=120)

    if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"\nCompleted with status: {state.runtime_status}")
        print(f"Result: {state.serialized_output}")
    elif state:
        print(f"\nOrchestration ended with status: {state.runtime_status}")
        if state.failure_details:
            print(f"Failure: {state.failure_details}")
    else:
        print("\nOrchestration did not complete within the timeout period")


if __name__ == "__main__":
    asyncio.run(main())
