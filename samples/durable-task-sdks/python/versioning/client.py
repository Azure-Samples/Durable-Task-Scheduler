import asyncio
import logging
import sys
import os
from azure.identity import DefaultAzureCredential, ManagedIdentityCredential
from durabletask.client import OrchestrationStatus
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main():
    """Main entry point for the client application."""
    logger.info("Starting Orchestration Versioning client...")
    
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
    
    # Create a client using Azure Managed Durable Task
    dts_client = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name,
        token_credential=credential
    )
    
    # Get name from command line or use default
    name = sys.argv[1] if len(sys.argv) > 1 else "World"
    
    # Define versions to test
    versions_to_test = [
        ("1.0.0", "v1 - Basic hello only"),
        ("2.0.0", "v2 - Hello + Goodbye"),
        ("3.0.0", "v3 - Hello + Goodbye + Notification"),
    ]
    
    try:
        logger.info(f"=== Orchestration Versioning Demo ===")
        logger.info(f"Testing with name: {name}")
        logger.info("")
        
        instance_ids = []
        
        # Schedule orchestrations with different versions
        for version, description in versions_to_test:
            logger.info(f"Scheduling orchestration with version {version}: {description}")
            
            instance_id = dts_client.schedule_new_orchestration(
                "versioned_orchestration",
                input=name,
                version=version
            )
            
            instance_ids.append((version, instance_id, description))
            logger.info(f"  Instance ID: {instance_id}")
        
        logger.info("")
        logger.info("Waiting for orchestrations to complete...")
        logger.info("")
        
        # Wait for all orchestrations to complete
        for version, instance_id, description in instance_ids:
            try:
                result = dts_client.wait_for_orchestration_completion(
                    instance_id,
                    timeout=60
                )
                
                if result:
                    if result.runtime_status == OrchestrationStatus.COMPLETED:
                        logger.info(f"=== Version {version} ({description}) ===")
                        logger.info(f"  Status: COMPLETED")
                        logger.info(f"  Result: {result.serialized_output}")
                    elif result.runtime_status == OrchestrationStatus.FAILED:
                        logger.error(f"=== Version {version} ===")
                        logger.error(f"  Status: FAILED")
                        logger.error(f"  Error: {result.failure_details}")
                    else:
                        logger.warning(f"=== Version {version} ===")
                        logger.warning(f"  Status: {result.runtime_status}")
                else:
                    logger.warning(f"=== Version {version} ===")
                    logger.warning(f"  Did not complete within timeout")
                    
            except Exception as e:
                logger.error(f"Error waiting for version {version}: {e}")
        
        logger.info("")
        logger.info("=== Demo Complete ===")
        logger.info("Key takeaway: All versions ran using the same worker code!")
        logger.info("Version gating in the orchestration ensures backward compatibility.")
            
    except KeyboardInterrupt:
        logger.info("Client stopped by user")
        
    except Exception as e:
        logger.exception(f"Unexpected error: {e}")
        
    finally:
        logger.info("Client shutting down")


if __name__ == "__main__":
    asyncio.run(main())
