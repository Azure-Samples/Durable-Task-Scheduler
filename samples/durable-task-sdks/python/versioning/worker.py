import asyncio
import logging
import os
from packaging import version
from azure.identity import DefaultAzureCredential, ManagedIdentityCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# Helper function to compare versions
def compare_version(v1: str | None, v2: str) -> int:
    """Compare two version strings.
    
    Returns:
        -1 if v1 < v2
         0 if v1 == v2
         1 if v1 > v2
    """
    if v1 is None:
        return -1
    try:
        ver1 = version.parse(v1)
        ver2 = version.parse(v2)
        if ver1 < ver2:
            return -1
        elif ver1 > ver2:
            return 1
        return 0
    except Exception:
        # Fall back to string comparison
        if v1 < v2:
            return -1
        elif v1 > v2:
            return 1
        return 0


# Activity functions
def say_hello(ctx: task.ActivityContext, name: str) -> str:
    """Activity that returns a greeting."""
    logger.info(f"Activity say_hello called with: {name}")
    return f"Hello, {name}!"


def say_goodbye(ctx: task.ActivityContext, name: str) -> str:
    """Activity added in v2.0.0 that says goodbye."""
    logger.info(f"Activity say_goodbye called with: {name}")
    return f"Goodbye, {name}!"


def send_notification(ctx: task.ActivityContext, message: str) -> str:
    """Activity added in v3.0.0 that sends a notification."""
    logger.info(f"Activity send_notification called with: {message}")
    return f"Notification sent: {message}"


# Versioned orchestrator function
def versioned_orchestration(ctx: task.OrchestrationContext, name: str):
    """Orchestration that demonstrates version-based branching.
    
    Version history:
    - v1.0.0: Basic hello greeting
    - v2.0.0: Added goodbye greeting
    - v3.0.0: Added notification after greeting
    
    The orchestration uses ctx.version to determine which code path to execute,
    ensuring backward compatibility for in-flight orchestrations.
    """
    results = []
    orch_version = ctx.version
    
    logger.info(f"Running orchestration with version: {orch_version}")
    
    # v1.0.0+: Basic hello greeting (all versions)
    hello_result = yield ctx.call_activity(say_hello, input=name)
    results.append(hello_result)
    
    # v2.0.0+: Added goodbye greeting
    if compare_version(orch_version, "2.0.0") >= 0:
        goodbye_result = yield ctx.call_activity(say_goodbye, input=name)
        results.append(goodbye_result)
    
    # v3.0.0+: Added notification
    if compare_version(orch_version, "3.0.0") >= 0:
        notification_message = f"Completed greeting workflow for {name}"
        notification_result = yield ctx.call_activity(send_notification, input=notification_message)
        results.append(notification_result)
    
    return {
        "version": orch_version,
        "results": results
    }


async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Orchestration Versioning worker...")
    
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
        worker.add_activity(say_goodbye)
        worker.add_activity(send_notification)
        worker.add_orchestrator(versioned_orchestration)
        
        # Start the worker
        worker.start()
        logger.info("Worker started. Listening for orchestrations...")
        
        try:
            # Keep the worker running
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("Worker shutdown initiated")
            
    logger.info("Worker stopped")


if __name__ == "__main__":
    asyncio.run(main())
