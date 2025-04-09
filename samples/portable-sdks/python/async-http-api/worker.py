import asyncio
import logging
import time
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Activity functions
def process_long_running_operation(ctx, data: dict) -> dict:
    """Activity that simulates a long-running operation."""
    logger.info(f"Processing long-running operation: {data}")
    operation_id = data.get("operation_id", "unknown")
    # Simulate a long-running process
    processing_time = data.get("processing_time", 5)
    logger.info(f"Operation {operation_id} will take {processing_time} seconds")
    
    # In a real-world scenario, this might be a call to an external service or system
    time.sleep(processing_time)
    
    return {
        "operation_id": operation_id,
        "status": "completed",
        "result": f"Operation {operation_id} completed successfully",
        "processed_at": time.time()
    }

# Orchestrator function
def async_http_api_orchestrator(ctx, input_data: dict) -> dict:
    """
    Orchestrator that demonstrates the async HTTP API pattern.
    
    This orchestrator starts a long-running operation and returns its result.
    """
    operation_id = input_data.get("operation_id", "unknown")
    logger.info(f"Starting async HTTP API orchestration for operation {operation_id}")
    
    # Execute the long-running operation
    result = yield ctx.call_activity("process_long_running_operation", input=input_data)
    
    logger.info(f"Completed orchestration for operation {operation_id}")
    return result

async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Async HTTP API pattern worker...")
    
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
    
    # Create a worker using Azure Managed Durable Task and start it with a context manager
    with DurableTaskSchedulerWorker(
        host_address=endpoint, 
        secure_channel=True,
        taskhub=taskhub_name, 
        token_credential=credential
    ) as worker:
        
        # Register activities and orchestrators
        worker.add_activity(process_long_running_operation)
        worker.add_orchestrator(async_http_api_orchestrator)
        
        # Start the worker (without awaiting)
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
