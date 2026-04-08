import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.extensions.azure_blob_payloads import BlobPayloadStore, BlobPayloadStoreOptions

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main():
    """Main entry point for the client application."""
    logger.info("Starting Large Payload pattern client...")

    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    # Azure Storage connection string (defaults to Azurite)
    storage_conn_str = os.getenv(
        "STORAGE_CONNECTION_STRING",
        "UseDevelopmentStorage=true",
    )

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")

    # Configure the blob payload store — must match the worker configuration
    store = BlobPayloadStore(BlobPayloadStoreOptions(
        connection_string=storage_conn_str,
        threshold_bytes=1_024,
    ))

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    client = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
        payload_store=store,
    )

    # --- Small payload (stays inline) ---
    print("\n--- Small payload (stays inline) ---")
    instance_id = client.schedule_new_orchestration(
        "large_payload_orchestrator", input=10
    )
    logger.info(f"Scheduled orchestration with ID: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=60)
    if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"Result: {state.serialized_output}")
    elif state:
        print(f"Orchestration failed: {state.failure_details}")

    # --- Large payload (externalized to blob storage) ---
    print("\n--- Large payload (externalized to blob storage) ---")
    instance_id = client.schedule_new_orchestration(
        "large_payload_orchestrator", input=10_000
    )
    logger.info(f"Scheduled orchestration with ID: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=60)
    if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"Result: {state.serialized_output}")
    elif state:
        print(f"Orchestration failed: {state.failure_details}")

    print("\nDone!")

if __name__ == "__main__":
    asyncio.run(main())
