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
    logger.info("Starting Function Chaining pattern client...")
    
    # Read the environment variable for taskhub
    taskhub_name = os.getenv("TASKHUB")
    if not taskhub_name:
        logger.error("TASKHUB is not set. Please set the TASKHUB environment variable.")
        logger.info("If you are using windows powershell, run: $env:TASKHUB=\"<taskhubname>\"")
        logger.info("If you are using bash, run: export TASKHUB=\"<taskhubname>\"")
        return
    
    # Read the environment variable
    endpoint = os.getenv("ENDPOINT")

    # Check if the variable exists
    if endpoint:
        print(f"The value of ENDPOINT is: {endpoint}")
    else:
        print("ENDPOINT is not set. Please set the ENDPOINT environment variable to the endpoint of the scheduler")
        print("If you are using windows powershell, run the following: $env:ENDPOINT=\"<schedulerEndpoint>\"")
        print("If you are using bash, run the following: export ENDPOINT=\"<schedulerEndpoint>\"")
        exit()
    
    credential = DefaultAzureCredential()
    
    # Create a client using Azure Managed Durable Task
    client = DurableTaskSchedulerClient(
        host_address=endpoint, 
        secure_channel=True,
        taskhub=taskhub_name, 
        token_credential=credential
    )
    
    # Get user input or use default name
    name = sys.argv[1] if len(sys.argv) > 1 else "User"
    
    logger.info(f"Starting new function chaining orchestration for {name}")
    
    # Schedule a new orchestration instance
    instance_id = client.schedule_new_orchestration(
        "function_chaining_orchestrator", 
        input=name
    )
    
    logger.info(f"Started orchestration with ID = {instance_id}")
    
    # Wait for orchestration to complete
    logger.info("Waiting for orchestration to complete...")
    result = client.wait_for_orchestration_completion(
        instance_id,
        timeout=30
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
