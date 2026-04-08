import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker
from durabletask.extensions.azure_blob_payloads import BlobPayloadStore, BlobPayloadStoreOptions

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# Activity functions
def generate_data(ctx: task.ActivityContext, num_records: int) -> str:
    """Activity that generates a payload of configurable size.

    For a small *num_records* value, the payload stays below the
    externalization threshold and is sent inline.  For a large value,
    the SDK automatically offloads the payload to Azure Blob Storage.
    """
    logger.info(f"Generating data with {num_records} records")
    return "RECORD|" * num_records


def process_data(ctx: task.ActivityContext, data: str) -> str:
    """Activity that summarizes the received data."""
    record_count = data.count("RECORD|")
    summary = f"Processed {record_count} records ({len(data)} bytes)"
    logger.info(summary)
    return summary


# Orchestrator function
def large_payload_orchestrator(ctx: task.OrchestrationContext, num_records: int):
    """Orchestrator that generates data and then processes it.

    Both the activity output (data) and the orchestration result are
    transparently externalized to blob storage when they exceed the
    configured threshold.
    """
    data = yield ctx.call_activity(generate_data, input=num_records)
    summary = yield ctx.call_activity(process_data, input=data)
    return summary


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Large Payload pattern worker...")

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

    # Configure the blob payload store with a low threshold for demo purposes
    store = BlobPayloadStore(BlobPayloadStoreOptions(
        connection_string=storage_conn_str,
        # Use a low threshold so that externalization is visible in the demo
        threshold_bytes=1_024,
    ))

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
        payload_store=store,
    ) as worker:

        # Register activities and orchestrator
        worker.add_activity(generate_data)
        worker.add_activity(process_data)
        worker.add_orchestrator(large_payload_orchestrator)

        # Start the worker
        worker.start()

        try:
            # Keep the worker running
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("Worker shutdown initiated")

    logger.info("Worker stopped")

if __name__ == "__main__":
    asyncio.run(main())
