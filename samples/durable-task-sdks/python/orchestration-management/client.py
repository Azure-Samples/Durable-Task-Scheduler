import asyncio
import logging
import os
from datetime import datetime, timezone
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main():
    """Main entry point demonstrating orchestration management operations."""
    logger.info("Starting Orchestration Management client...")

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

    # Record the start time for batch purge later
    start_time = datetime.now(timezone.utc)

    # =========================================================================
    # 1. Schedule and complete orchestrations
    # =========================================================================
    print("\n=== Step 1: Schedule orchestrations ===")
    instance_ids = []
    for i in range(3):
        instance_id = client.schedule_new_orchestration(
            "data_processing_orchestrator",
            input={"batch_id": f"batch-{i + 1}", "item_count": (i + 1) * 10},
        )
        instance_ids.append(instance_id)
        logger.info(f"Scheduled orchestration {i + 1} with ID: {instance_id}")

    # Wait for all orchestrations to complete
    for instance_id in instance_ids:
        state = client.wait_for_orchestration_completion(instance_id, timeout=60)
        if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
            print(f"  Completed: {instance_id} -> {state.serialized_output}")
        elif state:
            print(f"  Failed: {instance_id} -> {state.failure_details}")

    # =========================================================================
    # 2. Restart an orchestration (reuses the same instance ID)
    # =========================================================================
    print("\n=== Step 2: Restart orchestration (same instance ID) ===")
    original_id = instance_ids[0]
    restarted_id = client.restart_orchestration(original_id)
    print(f"  Restarted {original_id} -> new execution ID: {restarted_id}")

    state = client.wait_for_orchestration_completion(restarted_id, timeout=60)
    if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"  Restarted orchestration completed: {state.serialized_output}")

    # =========================================================================
    # 3. Restart with a new instance ID
    # =========================================================================
    print("\n=== Step 3: Restart orchestration (new instance ID) ===")
    new_id = client.restart_orchestration(
        instance_ids[1], restart_with_new_instance_id=True
    )
    print(f"  Restarted {instance_ids[1]} with new ID: {new_id}")

    state = client.wait_for_orchestration_completion(new_id, timeout=60)
    if state and state.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        print(f"  New orchestration completed: {state.serialized_output}")

    # =========================================================================
    # 4. Query orchestration instances
    # =========================================================================
    print("\n=== Step 4: Query orchestrations ===")
    query = durable_client.OrchestrationQuery(
        created_time_from=start_time,
        runtime_status=[durable_client.OrchestrationStatus.COMPLETED],
    )
    states = client.get_all_orchestration_states(query)
    print(f"  Found {len(states)} completed orchestration(s) since {start_time.isoformat()}")
    for s in states:
        print(f"    - {s.instance_id}: {s.name}")

    # =========================================================================
    # 5. Batch purge completed orchestrations
    # =========================================================================
    print("\n=== Step 5: Batch purge completed orchestrations ===")
    result = client.purge_orchestrations_by(
        created_time_from=start_time,
        runtime_status=[durable_client.OrchestrationStatus.COMPLETED],
    )
    print(f"  Purged {result.deleted_instance_count} orchestration(s)")

    # Verify purge worked
    states = client.get_all_orchestration_states(query)
    print(f"  Remaining completed orchestrations: {len(states)}")

    print("\nDone!")

if __name__ == "__main__":
    asyncio.run(main())
