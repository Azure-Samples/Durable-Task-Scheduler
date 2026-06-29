import asyncio
import logging
import os
import time
from datetime import datetime, timedelta, timezone
from azure.identity import DefaultAzureCredential
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.scheduled import (
    ScheduledTaskClient,
    ScheduleCreationOptions,
    ScheduleQuery,
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

SCHEDULE_ID = "report-every-5s"
# The schedule references the target orchestration by its registered name.
# The worker (worker.py) registers it as "report_orchestrator".
ORCHESTRATION_NAME = "report_orchestrator"


async def main():
    """Main entry point demonstrating recurring schedule management."""
    logger.info("Starting Scheduled Tasks client...")

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

    # The ScheduledTaskClient wraps the base client and exposes schedule
    # management operations (create, get, list, pause, resume, update, delete).
    scheduled_tasks = ScheduledTaskClient(client)

    # =========================================================================
    # 1. Create a recurring schedule
    # =========================================================================
    print("\n=== Step 1: Create a schedule ===")
    schedule = scheduled_tasks.create_schedule(ScheduleCreationOptions(
        schedule_id=SCHEDULE_ID,
        orchestration_name=ORCHESTRATION_NAME,
        interval=timedelta(seconds=5),
        orchestration_input="westus",
        start_at=datetime.now(timezone.utc),
        start_immediately_if_late=True,
    ))
    print(f"  Created schedule '{schedule.schedule_id}'")
    print(f"  Description: {scheduled_tasks.get_schedule(SCHEDULE_ID)}")

    # Let the schedule fire a few times.
    print("\n  Letting the schedule run for ~12 seconds...")
    time.sleep(12)

    # =========================================================================
    # 2. Pause and resume the schedule
    # =========================================================================
    print("\n=== Step 2: Pause and resume ===")
    schedule.pause()
    print("  Schedule paused")
    time.sleep(3)
    schedule.resume()
    print("  Schedule resumed")

    # =========================================================================
    # 3. List schedules by prefix
    # =========================================================================
    print("\n=== Step 3: List schedules ===")
    descriptions = scheduled_tasks.list_schedules(
        ScheduleQuery(schedule_id_prefix="report-")
    )
    print(f"  Found {len(descriptions)} schedule(s) with prefix 'report-':")
    for desc in descriptions:
        print(f"    - {desc.schedule_id}: status={desc.status}, next_run_at={desc.next_run_at}")

    # =========================================================================
    # 4. Delete the schedule
    # =========================================================================
    print("\n=== Step 4: Delete the schedule ===")
    schedule.delete()
    print(f"  Deleted schedule '{SCHEDULE_ID}'")
    print(f"  get_schedule now returns: {scheduled_tasks.get_schedule(SCHEDULE_ID)}")

    print("\nDone!")


if __name__ == "__main__":
    asyncio.run(main())
