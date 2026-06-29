import asyncio
import logging
import os
import time
from datetime import datetime, timedelta, timezone
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.extensions.history_export import (
    ExportDestination,
    ExportFormat,
    ExportFormatKind,
    ExportHistoryClient,
    ExportJobCreationOptions,
    ExportMode,
)
from durabletask.extensions.history_export.azure_blob import (
    AzureBlobHistoryExportWriter,
    AzureBlobHistoryExportWriterOptions,
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

CONTAINER_NAME = os.getenv("EXPORT_CONTAINER", "history-export-sample")
STORAGE_CONNECTION_STRING = os.getenv("STORAGE_CONNECTION_STRING", "UseDevelopmentStorage=true")


async def main():
    """Main entry point demonstrating orchestration history export."""
    logger.info("Starting History Export client...")

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

    # The client needs an ExportHistoryClient to create and poll export jobs.
    # The writer here mirrors the one configured on the worker; the worker's
    # activities perform the actual blob writes.
    writer = AzureBlobHistoryExportWriter(
        AzureBlobHistoryExportWriterOptions(
            container_name=CONTAINER_NAME,
            connection_string=STORAGE_CONNECTION_STRING,
        )
    )

    try:
        export_client = ExportHistoryClient(client, writer)

        # =====================================================================
        # 1. Seed some terminal orchestrations to export
        # =====================================================================
        print("\n=== Step 1: Seed sample orchestrations ===")
        for n in range(1, 6):
            instance_id = client.schedule_new_orchestration("sample_orchestrator", input=n)
            state = client.wait_for_orchestration_completion(instance_id, timeout=30)
            if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
                print(f"  Completed: {instance_id} -> {state.serialized_output}")
        # Give the scheduler a moment to finalize terminal history.
        time.sleep(1)

        # =====================================================================
        # 2. Create an export job for the recent time window
        # =====================================================================
        print("\n=== Step 2: Create export job ===")
        now = datetime.now(timezone.utc)
        desc = export_client.create_job(
            ExportJobCreationOptions(
                mode=ExportMode.BATCH,
                completed_time_from=now - timedelta(hours=1),
                completed_time_to=now + timedelta(hours=1),
                destination=ExportDestination(container=CONTAINER_NAME, prefix="sample-run"),
                format=ExportFormat(kind=ExportFormatKind.JSONL_GZIP),
                max_instances_per_batch=10,
            )
        )
        print(f"  job_id: {desc.job_id}")
        print(f"  orchestrator_instance_id: {desc.orchestrator_instance_id}")

        # =====================================================================
        # 3. Wait for the export job to finish and print the result
        # =====================================================================
        print("\n=== Step 3: Wait for export job ===")
        final = export_client.wait_for_job(desc.job_id, timeout=120, poll_interval=0.5)
        print(f"  status:             {final.status.value}")
        print(f"  scanned_instances:  {final.scanned_instances}")
        print(f"  exported_instances: {final.exported_instances}")
        print(f"  failed_instances:   {final.failed_instances}")
        if final.last_error:
            print(f"  last_error:         {final.last_error}")

        print("\nDone!")
    finally:
        writer.close()


if __name__ == "__main__":
    asyncio.run(main())
