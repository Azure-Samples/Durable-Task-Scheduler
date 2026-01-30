import asyncio
import logging
import os
from azure.identity import DefaultAzureCredential, ManagedIdentityCredential
from azure.core.exceptions import ClientAuthenticationError
from durabletask import task, entities
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# Function-based entity for a counter
def counter(ctx: entities.EntityContext, input: int):
    """Function-based entity that maintains a counter state.
    
    Supports operations: add, subtract, get, reset
    """
    state = ctx.get_state(int, 0)  # Get state with default 0
    
    if ctx.operation == "add":
        state += input
        ctx.set_state(state)
        logger.info(f"Counter '{ctx.entity_id.key}': Added {input}, new value: {state}")
    elif ctx.operation == "subtract":
        state -= input
        ctx.set_state(state)
        logger.info(f"Counter '{ctx.entity_id.key}': Subtracted {input}, new value: {state}")
    elif ctx.operation == "get":
        logger.info(f"Counter '{ctx.entity_id.key}': Current value: {state}")
        return state
    elif ctx.operation == "reset":
        ctx.set_state(0)
        logger.info(f"Counter '{ctx.entity_id.key}': Reset to 0")


# Orchestrator that interacts with the counter entity
def counter_workflow(ctx: task.OrchestrationContext, entity_key: str):
    """Orchestration that demonstrates entity interactions.
    
    This orchestration:
    1. Creates/accesses a counter entity
    2. Adds values to the counter
    3. Gets the current value
    4. Subtracts a value
    5. Returns the final count
    """
    entity_id = entities.EntityInstanceId("counter", entity_key)
    
    # Signal entity operations (fire-and-forget)
    ctx.signal_entity(entity_id=entity_id, operation_name="add", input=10)
    ctx.signal_entity(entity_id=entity_id, operation_name="add", input=5)
    ctx.signal_entity(entity_id=entity_id, operation_name="subtract", input=3)
    
    # Call entity and wait for result (note: call_entity uses 'entity' and 'operation' params)
    value = yield ctx.call_entity(entity=entity_id, operation="get")
    
    return f"Counter '{entity_key}' final value: {value}"


# Activity to log entity state
def log_entity_state(ctx: task.ActivityContext, message: str) -> str:
    """Activity function that logs messages."""
    logger.info(f"Entity state log: {message}")
    return message


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Entities pattern worker...")
    
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
        
        # Register entities, activities and orchestrators
        worker.add_entity(counter)
        worker.add_activity(log_entity_state)
        worker.add_orchestrator(counter_workflow)
        
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
