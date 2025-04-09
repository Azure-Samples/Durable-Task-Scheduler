import asyncio
import logging
import random
import time
import os
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Activity function
def process_work_item(ctx, item: int) -> dict:
    """
    Activity function that processes a single work item.
    
    This simulates processing a single item with some random delay.
    """
    logger.info(f"Processing work item: {item}")
    # Simulate processing work that takes random time
    time.sleep(random.uniform(0.5, 2.0))
    result = item * item
    return {"item": item, "result": result}

def aggregate_results(ctx, results: list) -> dict:
    """
    Activity function that aggregates the results from multiple work items.
    """
    logger.info(f"Aggregating results from {len(results)} items")
    sum_result = sum(item["result"] for item in results)
    return {
        "total_items": len(results),
        "sum": sum_result,
        "average": sum_result / len(results) if results else 0
    }

# Orchestrator function
def fan_out_fan_in_orchestrator(ctx, work_items: list) -> dict:
    """
    Orchestrator that demonstrates fan out/fan in pattern.
    
    This orchestrator processes multiple items in parallel (fan out) and then
    aggregates the results once all parallel executions complete (fan in).
    """
    logger.info(f"Starting fan out/fan in orchestration with {len(work_items)} items")
    
    # Fan out: Create a task for each work item
    parallel_tasks = []
    for item in work_items:
        parallel_tasks.append(ctx.call_activity("process_work_item", input=item))
    
    # Wait for all tasks to complete
    logger.info(f"Waiting for {len(parallel_tasks)} parallel tasks to complete")
    results = yield task.when_all(parallel_tasks)
    
    # Fan in: Aggregate all the results
    logger.info("All parallel tasks completed, aggregating results")
    final_result = yield ctx.call_activity("aggregate_results", input=results)
    
    return final_result

async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Fan Out/Fan In pattern worker...")
    
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
        worker.add_activity(process_work_item)
        worker.add_activity(aggregate_results)
        worker.add_orchestrator(fan_out_fan_in_orchestrator)
        
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
