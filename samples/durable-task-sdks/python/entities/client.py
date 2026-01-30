import asyncio
import logging
import sys
import os
from azure.identity import DefaultAzureCredential, ManagedIdentityCredential
from azure.core.exceptions import ClientAuthenticationError
from durabletask import client as durable_client, entities
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main():
    """Main entry point for the client application."""
    logger.info("Starting Entities pattern client...")
    
    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    # Default interval between orchestrations (in seconds)
    interval = int(os.getenv("ORCHESTRATION_INTERVAL", "60"))

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")
    print(f"Orchestration interval: {interval} seconds")

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
    
    # Create a client using Azure Managed Durable Task
    logger.info(f"Creating client with endpoint={endpoint}, credential={'provided' if credential else 'none'}")
    client = DurableTaskSchedulerClient(
        host_address=endpoint, 
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name, 
        token_credential=credential
    )
    
    # Get entity key from command line or use default
    entity_key = sys.argv[1] if len(sys.argv) > 1 else "my-counter"
    
    # Set up orchestration parameters
    TOTAL_ORCHESTRATIONS = 5  # Total number of orchestrations to run
    INTERVAL_SECONDS = 5       # Time between orchestrations in seconds
    completed_orchestrations = 0
    failed_orchestrations = 0
    
    try:
        logger.info(f"Starting entity operations demo - {TOTAL_ORCHESTRATIONS} orchestrations")
        
        # First, demonstrate direct entity signaling from client
        logger.info("=== Direct Entity Operations ===")
        entity_id = entities.EntityInstanceId("counter", entity_key)
        
        # Signal the entity directly from the client
        logger.info(f"Signaling entity '{entity_key}' to add 100")
        client.signal_entity(entity_id, "add", input=100)
        
        # Wait a moment for the signal to be processed
        await asyncio.sleep(2)
        
        logger.info(f"Signaling entity '{entity_key}' to subtract 25")
        client.signal_entity(entity_id, "subtract", input=25)
        
        await asyncio.sleep(2)
        
        # Now run orchestrations that interact with entities
        logger.info("=== Orchestration-based Entity Operations ===")
        
        instance_ids = []
        for i in range(TOTAL_ORCHESTRATIONS):
            try:
                # Create a unique entity key for this orchestration
                instance_entity_key = f"{entity_key}-orch-{i+1}"
                logger.info(f"Scheduling orchestration #{i+1} for entity '{instance_entity_key}'")
                
                # Schedule the orchestration
                instance_id = client.schedule_new_orchestration(
                    "counter_workflow",
                    input=instance_entity_key
                )
                
                instance_ids.append(instance_id)
                logger.info(f"Orchestration #{i+1} scheduled with ID: {instance_id}")
                
                # Wait before scheduling next orchestration (except for the last one)
                if i < TOTAL_ORCHESTRATIONS - 1:
                    logger.info(f"Waiting {INTERVAL_SECONDS} seconds before scheduling next orchestration...")
                    await asyncio.sleep(INTERVAL_SECONDS)
                
            except Exception as e:
                logger.error(f"Error scheduling orchestration #{i+1}: {e}")
        
        logger.info(f"All {len(instance_ids)} orchestrations scheduled. Waiting for completion...")
        
        # Wait for all orchestrations to complete
        for idx, instance_id in enumerate(instance_ids):
            try:
                logger.info(f"Waiting for orchestration {idx+1}/{len(instance_ids)} (ID: {instance_id})...")
                result = client.wait_for_orchestration_completion(
                    instance_id,
                    timeout=120
                )
                
                if result:
                    if result.runtime_status == durable_client.OrchestrationStatus.FAILED:
                        failed_orchestrations += 1
                        logger.error(f"Orchestration {instance_id} failed")
                    elif result.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
                        completed_orchestrations += 1
                        logger.info(f"Orchestration {instance_id} completed successfully with result: {result.serialized_output}")
                    else:
                        logger.info(f"Orchestration {instance_id} status: {result.runtime_status}")
                else:
                    logger.warning(f"Orchestration {instance_id} did not complete within the timeout period")
            except Exception as e:
                logger.error(f"Error waiting for orchestration {instance_id}: {e}")
        
        logger.info(f"All orchestrations processed. Successful: {completed_orchestrations}, Failed: {failed_orchestrations}")
        
        # Final summary
        logger.info("=== Entity Demo Complete ===")
        logger.info(f"Direct entity signals sent to '{entity_key}'")
        logger.info(f"Orchestrations completed: {completed_orchestrations}/{TOTAL_ORCHESTRATIONS}")
            
    except KeyboardInterrupt:
        logger.info("Client stopped by user")
        
    except Exception as e:
        logger.exception(f"Unexpected error: {e}")
        
    finally:
        logger.info("Client shutting down")


if __name__ == "__main__":
    asyncio.run(main())
