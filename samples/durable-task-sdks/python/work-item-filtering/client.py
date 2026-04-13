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
    """Main entry point for the client application."""
    logger.info("Starting Work Item Filtering client...")

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

    # --- Schedule a greeting orchestration (handled by Worker A) ---
    print("\n--- Scheduling greeting orchestration (Worker A) ---")
    greeting_id = client.schedule_new_orchestration(
        "greeting_orchestrator", input="World"
    )
    logger.info(f"Greeting orchestration scheduled with ID: {greeting_id}")

    # --- Schedule a math orchestration (handled by Worker B) ---
    print("\n--- Scheduling math orchestration (Worker B) ---")
    math_id = client.schedule_new_orchestration(
        "math_orchestrator", input={"a": 40, "b": 2}
    )
    logger.info(f"Math orchestration scheduled with ID: {math_id}")

    # --- Wait for both to complete ---
    print("\nWaiting for orchestrations to complete...")

    greeting_state = client.wait_for_orchestration_completion(greeting_id, timeout=60)
    if greeting_state and greeting_state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"Greeting result: {greeting_state.serialized_output}")
    elif greeting_state:
        print(f"Greeting orchestration failed: {greeting_state.failure_details}")

    math_state = client.wait_for_orchestration_completion(math_id, timeout=60)
    if math_state and math_state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"Math result: {math_state.serialized_output}")
    elif math_state:
        print(f"Math orchestration failed: {math_state.failure_details}")

    print("\nDone! Check the worker terminal outputs to see which worker handled each orchestration.")

if __name__ == "__main__":
    asyncio.run(main())
