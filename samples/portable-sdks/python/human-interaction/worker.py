import asyncio
import datetime
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
def submit_approval_request(ctx, request_data: dict) -> dict:
    """
    Activity that simulates submitting an approval request.
    In a real application, this would notify a human approver via email, message, etc.
    """
    request_id = request_data.get("request_id")
    requester = request_data.get("requester")
    item = request_data.get("item")
    
    logger.info(f"Submitting approval request {request_id} from {requester} for {item}")
    
    # In a real system, this would send an email, notification, or update a database
    return {
        "request_id": request_id,
        "status": "Pending",
        "submitted_at": datetime.datetime.now().isoformat(),
        "approval_url": f"http://localhost:8000/api/approvals/{request_id}"
    }

def process_approval(ctx, approval_data: dict) -> dict:
    """
    Activity that processes the approval once received.
    """
    request_id = approval_data.get("request_id")
    is_approved = approval_data.get("is_approved")
    approver = approval_data.get("approver")
    
    approval_status = "Approved" if is_approved else "Rejected"
    logger.info(f"Processing {approval_status} request {request_id} by {approver}")
    
    # In a real system, this would update a database, trigger workflows, etc.
    return {
        "request_id": request_id,
        "status": approval_status,
        "processed_at": datetime.datetime.now().isoformat(),
        "approver": approver
    }

# Orchestrator function
def human_interaction_orchestrator(ctx, input_data: dict) -> dict:
    """
    Orchestrator that demonstrates the human interaction pattern.
    
    This orchestrator submits an approval request, then waits for a human
    to approve or reject before continuing.
    """
    request_id = input_data.get("request_id")
    requester = input_data.get("requester")
    item = input_data.get("item")
    timeout_hours = input_data.get("timeout_hours", 24)
    
    logger.info(f"Starting human interaction orchestration for request {request_id}")
    
    # Submit the approval request
    request_data = {
        "request_id": request_id,
        "requester": requester,
        "item": item
    }
    
    submission_result = yield ctx.call_activity("submit_approval_request", input=request_data)
    
    # Make the status available via custom status
    ctx.set_custom_status(submission_result)
    
    # Create a durable timer for the timeout
    timeout_deadline = ctx.current_utc_datetime + datetime.timedelta(hours=timeout_hours)
    timeout_task = ctx.create_timer(timeout_deadline)
    
    # Wait for an external event (approval/rejection) or timeout
    approval_event_name = "approval_response"
    
    # Create a task that waits for the external event
    approval_task = ctx.wait_for_external_event(approval_event_name)
    
    # Wait for either the timeout or the approval response, whichever comes first
    winner_task = yield task.when_any([approval_task, timeout_task])
    
    # Process based on which task completed
    result = {}
    if winner_task == approval_task:
        # Human responded in time
        # Get the event result - in the new SDK, we need to access the output of the task properly
        approval_data = yield approval_task
        logger.info(f"Received approval response for request {request_id}")
        
        # Process the approval
        result = yield ctx.call_activity("process_approval", input={
            "request_id": request_id,
            "is_approved": approval_data.get("is_approved", False),
            "approver": approval_data.get("approver", "Unknown")
        })
    else:
        # Timeout occurred
        logger.info(f"Request {request_id} timed out waiting for approval")
        result = {
            "request_id": request_id,
            "status": "Timeout",
            "timed_out_at": ctx.current_utc_datetime.isoformat()
        }
    
    return result

async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Human Interaction pattern worker...")
    
    # Get environment variables for taskhub and endpoint with defaults
    taskhub_name = os.getenv("TASKHUB", "default")
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    print(f"Using taskhub: {taskhub_name}")
    print(f"Using endpoint: {endpoint}")

    # Set credential to None for emulator, or DefaultAzureCredential for Azure
    credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()
    
    # Create a worker using Azure Managed Durable Task and start it with a context manager
    with DurableTaskSchedulerWorker(
        host_address=endpoint, 
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub_name, 
        token_credential=credential
    ) as worker:
        
        # Register activities and orchestrators
        worker.add_activity(submit_approval_request)
        worker.add_activity(process_approval)
        worker.add_orchestrator(human_interaction_orchestrator)
        
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
