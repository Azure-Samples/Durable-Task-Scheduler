import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential, ManagedIdentityCredential
from azure.core.exceptions import ClientAuthenticationError
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
    
    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")
    
    # Credential handling with better error management
    credential = None
    if endpoint != "http://localhost:8080":
        try:
            # Check if we're running in Azure with a managed identity
            client_id = os.getenv("AZURE_MANAGED_IDENTITY_CLIENT_ID")
            if client_id:
                logger.info(f"Using Managed Identity with client ID: {client_id}")
                credential = ManagedIdentityCredential(client_id=client_id)
                # Test the credential to make sure it works
                credential.get_token("https://management.azure.com/.default")
                logger.info("Successfully authenticated with Managed Identity")
            else:
                # Fall back to DefaultAzureCredential only if no client ID is available
                logger.info("No client ID found, falling back to DefaultAzureCredential")
                credential = DefaultAzureCredential()
        except Exception as e:
            logger.error(f"Authentication error: {e}")
            logger.warning("Continuing without authentication - this may only work with local emulator")
            credential = None
    
    with DurableTaskSchedulerWorker(
        host_address=endpoint, 
        secure_channel=endpoint != "http://localhost:8080",
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
