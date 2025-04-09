import asyncio
import logging
import sys
import os
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def main():
    """Main entry point for the client application."""
    logger.info("Starting Fan Out/Fan In pattern client...")
    
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
    
    # Generate work items (default 10 items if not specified)
    count = int(sys.argv[1]) if len(sys.argv) > 1 else 10
    work_items = list(range(1, count + 1))
    
    logger.info(f"Starting new fan out/fan in orchestration with {count} work items")
    
    # Schedule a new orchestration instance
    instance_id = client.schedule_new_orchestration(
        "fan_out_fan_in_orchestrator", 
        input=work_items
    )
    
    logger.info(f"Started orchestration with ID = {instance_id}")
    
    # Wait for orchestration to complete
    logger.info("Waiting for orchestration to complete...")
    result = client.wait_for_orchestration_completion(
        instance_id,
        timeout=60
    )
    
    if result:
        if result.runtime_status == durable_client.OrchestrationStatus.FAILED:
            logger.error(f"Orchestration failed")
        elif result.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
            logger.info(f"Orchestration completed with result: {result.serialized_output}")
        else:
            logger.info(f"Orchestration status: {result.runtime_status}")
    else:
        logger.warning("Orchestration did not complete within the timeout period")

if __name__ == "__main__":
    asyncio.run(main())
