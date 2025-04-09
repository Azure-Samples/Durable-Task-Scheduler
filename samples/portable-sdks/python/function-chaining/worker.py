import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Activity functions
def say_hello(ctx, name: str) -> str:
    """First activity that greets the user."""
    logger.info(f"Activity say_hello called with name: {name}")
    return f"Hello {name}!"

def process_greeting(ctx, greeting: str) -> str:
    """Second activity that processes the greeting."""
    logger.info(f"Activity process_greeting called with greeting: {greeting}")
    return f"{greeting} How are you today?"

def finalize_response(ctx, response: str) -> str:
    """Third activity that finalizes the response."""
    logger.info(f"Activity finalize_response called with response: {response}")
    return f"{response} I hope you're doing well!"

# Orchestrator function
def function_chaining_orchestrator(ctx, name: str) -> str:
    """Orchestrator that demonstrates function chaining pattern."""
    logger.info(f"Starting function chaining orchestration for {name}")
    
    # Call first activity - passing input directly without named parameter
    greeting = yield ctx.call_activity('say_hello', input=name)
    
    # Call second activity with the result from first activity
    processed_greeting = yield ctx.call_activity('process_greeting', input=greeting)
    
    # Call third activity with the result from second activity
    final_response = yield ctx.call_activity('finalize_response', input=processed_greeting)
    
    return final_response

async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Function Chaining pattern worker...")
    
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
    
    # Split the endpoint if it contains authentication info
    if ";" in endpoint:
        host_address = endpoint.split(';')[0]
    else:
        host_address = endpoint
    
    credential = DefaultAzureCredential()
    
    # Create a worker using Azure Managed Durable Task and start it with a context manager
    with DurableTaskSchedulerWorker(
        host_address=host_address, 
        secure_channel=True,
        taskhub=taskhub_name, 
        token_credential=credential
    ) as worker:
        
        # Register activities and orchestrators
        worker.add_activity(say_hello)
        worker.add_activity(process_greeting)
        worker.add_activity(finalize_response)
        worker.add_orchestrator(function_chaining_orchestrator)
        
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
