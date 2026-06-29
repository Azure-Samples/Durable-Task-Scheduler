import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker
from durabletask.extensions.history_export import ExportHistoryClient
from durabletask.extensions.history_export.azure_blob import (
    AzureBlobHistoryExportWriter,
    AzureBlobHistoryExportWriterOptions,
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Destination container and storage connection string. Defaults target Azurite,
# the local Azure Storage emulator. Set STORAGE_CONNECTION_STRING to point at a
# real Azure Storage account instead.
CONTAINER_NAME = os.getenv("EXPORT_CONTAINER", "history-export-sample")
STORAGE_CONNECTION_STRING = os.getenv("STORAGE_CONNECTION_STRING", "UseDevelopmentStorage=true")


# Sample workload that produces orchestration history to export
def square(ctx: task.ActivityContext, n: int) -> int:
    """Activity that squares a number."""
    logger.info(f"Squaring {n}")
    return n * n


def sample_orchestrator(ctx: task.OrchestrationContext, n: int):
    """Simple orchestration whose history will be exported."""
    result = yield ctx.call_activity(square, input=n)
    return result


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting History Export pattern worker...")

    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")
    print(f"Using container: {CONTAINER_NAME}")

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

    client = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential,
    )

    # The Azure Blob writer is what the export activities use to write each
    # instance's history as gzipped JSONL to the destination container.
    writer = AzureBlobHistoryExportWriter(
        AzureBlobHistoryExportWriterOptions(
            container_name=CONTAINER_NAME,
            connection_string=STORAGE_CONNECTION_STRING,
        )
    )

    try:
        export_client = ExportHistoryClient(client, writer)

        with DurableTaskSchedulerWorker(
            host_address=endpoint,
            secure_channel=endpoint != "http://localhost:8080",
            taskhub=taskhub_name,
            token_credential=credential,
        ) as worker:

            # Register the sample workload
            worker.add_activity(square)
            worker.add_orchestrator(sample_orchestrator)

            # Register the export-job entity, activities, and orchestrator that
            # perform the actual history export. This binds the writer above so
            # the export activities can write to blob storage.
            export_client.register_worker(worker)

            worker.start()

            try:
                while True:
                    await asyncio.sleep(1)
            except KeyboardInterrupt:
                logger.info("Worker shutdown initiated")
    finally:
        writer.close()

    logger.info("Worker stopped")

if __name__ == "__main__":
    asyncio.run(main())
