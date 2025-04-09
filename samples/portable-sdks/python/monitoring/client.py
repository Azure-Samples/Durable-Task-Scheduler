import asyncio
import logging
import sys
import uuid
import os
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def main():
    """Main entry point for the client application."""
    logger.info("Starting Monitoring pattern client...")
    
    # Read the environment variable for taskhub
    taskhub_name = os.getenv("TASKHUB")
    if not taskhub_name:
        logger.error("TASKHUB is not set. Please set the TASKHUB environment variable.")
        return
    
    # Read the environment variable for endpoint
    endpoint = os.getenv("ENDPOINT")
    if not endpoint:
        logger.error("ENDPOINT is not set. Please set the ENDPOINT environment variable.")
        return
    
    credential = DefaultAzureCredential()
    
    # Create a client using Azure Managed Durable Task
    client = DurableTaskSchedulerClient(
        host_address=endpoint, 
        secure_channel=True,
        taskhub=taskhub_name, 
        token_credential=credential
    )
    
    # Generate a unique job ID or use one provided as an argument
    job_id = sys.argv[1] if len(sys.argv) > 1 else f"job-{uuid.uuid4()}"
    
    # Define monitoring parameters
    polling_interval = int(sys.argv[2]) if len(sys.argv) > 2 else 5  # seconds
    timeout = int(sys.argv[3]) if len(sys.argv) > 3 else 30  # seconds
    
    job_data = {
        "job_id": job_id,
        "polling_interval_seconds": polling_interval,
        "timeout_seconds": timeout
    }
    
    logger.info(f"Starting monitoring for job {job_id}")
    logger.info(f"Polling interval: {polling_interval} seconds")
    logger.info(f"Timeout: {timeout} seconds")
    
    # Schedule a new orchestration instance
    instance_id = client.schedule_new_orchestration(
        "monitoring_job_orchestrator", 
        input=job_data
    )
    
    logger.info(f"Started monitoring orchestration with ID = {instance_id}")
    
    # Wait for orchestration to complete while showing updates
    logger.info("Waiting for monitoring to complete...")
    logger.info("Status updates will be displayed as they occur.")
    
    # Create a simple timeout for the demonstration
    max_wait_time = timeout + 10  # Add a buffer to the timeout
    
    last_status = None
    total_wait_time = 0
    wait_interval = 2  # seconds
    
    while total_wait_time < max_wait_time:
        # Get the current orchestration state
        state = client.get_orchestration_state(instance_id)
        
        if state:
            # Display custom status updates if available and different from last update
            # Use serialized_custom_status instead of custom_status with proper parsing
            if hasattr(state, 'serialized_custom_status') and state.serialized_custom_status:
                import json
                try:
                    current_status = json.loads(state.serialized_custom_status)
                    if current_status != last_status:
                        last_status = current_status
                        logger.info(f"Status update: {last_status}")
                except json.JSONDecodeError:
                    logger.warning("Could not parse custom status as JSON")
            
            # Check if the orchestration has completed
            if state.runtime_status in (durable_client.OrchestrationStatus.COMPLETED, 
                                       durable_client.OrchestrationStatus.FAILED, 
                                       durable_client.OrchestrationStatus.TERMINATED):
                logger.info(f"Monitoring completed with status: {state.runtime_status}")
                
                if state.runtime_status == "Completed":
                    logger.info(f"Final result: {state.output}")
                elif state.runtime_status == "Failed":
                    logger.error(f"Monitoring failed: {state.failure_details}")
                
                break
        
        # Wait before checking again
        await asyncio.sleep(wait_interval)
        total_wait_time += wait_interval
    
    if total_wait_time >= max_wait_time:
        logger.warning("Client timed out waiting for orchestration to complete")

if __name__ == "__main__":
    asyncio.run(main())
