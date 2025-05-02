import asyncio
import logging
import uuid
import os
import datetime
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Get environment variables for taskhub and endpoint
TASKHUB = os.getenv("TASKHUB", "default")
ENDPOINT = os.getenv("ENDPOINT", "http://localhost:8080")

# Check if environment variables are set
if not TASKHUB:
    logger.error("TASKHUB is not set. Please set the TASKHUB environment variable.")

if not ENDPOINT:
    logger.error("ENDPOINT is not set. Please set the ENDPOINT environment variable.")

async def get_client():
    """Get a Durable Task client."""
    credential = DefaultAzureCredential()
    return DurableTaskSchedulerClient(
        host_address=ENDPOINT, 
        secure_channel=True,
        taskhub=TASKHUB, 
        token_credential=credential
    )

async def create_approval_request(requester, item, timeout_hours=24.0):
    """Create a new approval request."""
    # Generate a unique request ID
    request_id = str(uuid.uuid4())
    logger.info(f"Creating new approval request with ID: {request_id}")
    
    # Get client
    client = await get_client()
    
    # Prepare input for the orchestration
    input_data = {
        "request_id": request_id,
        "requester": requester,
        "item": item,
        "timeout_hours": timeout_hours
    }
    
    # Schedule the orchestration
    instance_id = client.schedule_new_orchestration(
        "human_interaction_orchestrator", 
        input=input_data,
        instance_id=request_id  # Use request_id as instance_id for simplicity
    )
    
    logger.info(f"Orchestration scheduled with ID: {instance_id}")
    return request_id

async def get_request_status(request_id):
    """Get the status of an approval request."""
    logger.info(f"Checking status for request: {request_id}")
    
    # Get client
    client = await get_client()
    
    # Get the orchestration status
    status = client.get_orchestration_state(request_id)
    
    if not status:
        logger.error(f"Request {request_id} not found")
        return None
    
    result = {
        "request_id": request_id,
        "runtime_status": status.runtime_status
    }
    
    # Add custom status if available
    if hasattr(status, 'serialized_custom_status') and status.serialized_custom_status:
        import json
        try:
            if isinstance(status.serialized_custom_status, str):
                custom_status = json.loads(status.serialized_custom_status)
                result["details"] = custom_status
            else:
                result["details"] = status.serialized_custom_status
        except json.JSONDecodeError:
            result["details"] = status.serialized_custom_status
    
    # Add output if completed
    if status.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        import json
        try:
            if hasattr(status, 'serialized_output') and status.serialized_output:
                result["output"] = json.loads(status.serialized_output)
            else:
                result["output"] = None
        except json.JSONDecodeError:
            result["output"] = status.serialized_output
    elif status.runtime_status == durable_client.OrchestrationStatus.FAILED:
        result["error"] = str(status.failure_details)
    
    return result

async def respond_to_approval(request_id, is_approved=True, approver="Console User", comments=""):
    """Respond to an approval request."""
    logger.info(f"Processing {approver}'s response to request {request_id}")
    
    # Get client
    client = await get_client()
    
    # Check if the orchestration exists and is running
    status = client.get_orchestration_state(request_id)
    
    if not status:
        logger.error(f"Request {request_id} not found")
        return False
    
    # Check if the orchestration is in a state where we can respond to it
    if status.runtime_status not in [
        durable_client.OrchestrationStatus.RUNNING, 
        durable_client.OrchestrationStatus.PENDING
    ]:
        logger.error(f"Cannot respond to request with status {status.runtime_status}")
        return False
    
    # Send the approval response as an external event
    approval_data = {
        "is_approved": is_approved,
        "approver": approver,
        "comments": comments,
        "response_time": datetime.datetime.now().isoformat()
    }
    
    client.raise_orchestration_event(
        instance_id=request_id, 
        event_name="approval_response", 
        data=approval_data
    )
    
    approval_status = "approved" if is_approved else "rejected"
    logger.info(f"Request {approval_status} by {approver}")
    return True

async def print_status_details(status):
    """Print readable status details."""
    if not status:
        print("  Status: Not found")
        return
        
    print(f"  Status: {status['runtime_status']}")
    
    if "details" in status:
        print(f"  Details: {status['details']}")
        
    if "output" in status:
        print(f"  Output: {status['output']}")
        
    if "error" in status:
        print(f"  Error: {status['error']}")

async def main():
    """Main entry point for console application."""
    print("=== Human Interaction Pattern Console Client ===")
    print("This sample demonstrates the human interaction pattern with a console-based workflow.")
    
    # Create a request
    print("\nCreating a new approval request...")
    request_id = await create_approval_request(
        requester="Console User",
        item="Vacation Request",
        timeout_hours=1  # Short timeout for demonstration
    )
    
    print(f"\nRequest created with ID: {request_id}")
    
    # Initial status check
    print("\nChecking initial status...")
    status = await get_request_status(request_id)
    await print_status_details(status)
    
    # Wait for user to decide
    print("\nPress Enter to approve the request, or type 'reject' and press Enter to reject: ")
    user_input = input().strip().lower()
    is_approved = user_input != "reject"
    
    # Process the response
    print("\nSubmitting your response...")
    success = await respond_to_approval(
        request_id=request_id,
        is_approved=is_approved,
        comments="Response from console application"
    )
    
    if not success:
        print("Failed to submit response")
        return
    
    # Check final status
    print("\nWaiting for final status...")
    for _ in range(3):  # Check a few times with delay
        await asyncio.sleep(2)
        status = await get_request_status(request_id)
        if status and status["runtime_status"] == "COMPLETED":
            break
    
    print("\nFinal status:")
    await print_status_details(status)
    
    print("\nSample completed.")

if __name__ == "__main__":
    # Run the async main function
    asyncio.run(main())

