import asyncio
import logging
import uuid
import os
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from azure.identity import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Set up FastAPI app
app = FastAPI(title="Durable Task Human Interaction Sample")

# Models for request and response
class ApprovalRequest(BaseModel):
    requester: str
    item: str
    timeout_hours: float = 24.0  # Default timeout in hours

class ApprovalResponse(BaseModel):
    is_approved: bool
    approver: str
    comments: str = ""

# Dictionary to store client references
client_cache = {}

# Get environment variables for taskhub and endpoint
TASKHUB = os.getenv("TASKHUB")
ENDPOINT = os.getenv("ENDPOINT")

# Check if environment variables are set
if not TASKHUB:
    logger.error("TASKHUB is not set. Please set the TASKHUB environment variable.")

if not ENDPOINT:
    logger.error("ENDPOINT is not set. Please set the ENDPOINT environment variable.")

async def get_client():
    """Get or create a Durable Task client."""
    if "client" not in client_cache and TASKHUB and ENDPOINT:
        credential = DefaultAzureCredential()
        client_cache["client"] = DurableTaskSchedulerClient(
            host_address=ENDPOINT, 
            secure_channel=True,
            taskhub=TASKHUB, 
            token_credential=credential
        )
    return client_cache["client"]

@app.post("/api/requests")
async def create_approval_request(request: ApprovalRequest):
    """Create a new approval request."""
    # Generate a unique request ID
    request_id = str(uuid.uuid4())
    logger.info(f"Creating new approval request with ID: {request_id}")
    
    # Get client
    client = await get_client()
    
    # Prepare input for the orchestration
    input_data = {
        "request_id": request_id,
        "requester": request.requester,
        "item": request.item,
        "timeout_hours": request.timeout_hours
    }
    
    # Schedule the orchestration
    instance_id = client.schedule_new_orchestration(
        "human_interaction_orchestrator", 
        input=input_data,
        instance_id=request_id  # Use request_id as instance_id for simplicity
    )
    
    return {
        "request_id": request_id,
        "status": "Pending",
        "approval_url": f"/api/approvals/{request_id}",
        "status_url": f"/api/requests/{request_id}"
    }

@app.get("/api/requests/{request_id}")
async def get_request_status(request_id: str):
    """Get the status of an approval request."""
    logger.info(f"Checking status for request: {request_id}")
    
    # Get client
    client = await get_client()
    
    # Get the orchestration status
    status = client.get_orchestration_state(request_id)
    
    if not status:
        raise HTTPException(status_code=404, detail=f"Request {request_id} not found")
    
    result = {
        "request_id": request_id,
        "runtime_status": status.runtime_status
    }
    
    # Add custom status if available, using hasattr to safely check attribute existence
    if hasattr(status, 'serialized_custom_status') and status.serialized_custom_status:
        # Parse the JSON string if it's a string
        if isinstance(status.serialized_custom_status, str):
            import json
            try:
                custom_status = json.loads(status.serialized_custom_status)
                result["details"] = custom_status
            except json.JSONDecodeError:
                result["details"] = status.serialized_custom_status
        else:
            result["details"] = status.serialized_custom_status
    
    # Add output if completed, using serialized_output instead of output
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

@app.post("/api/approvals/{request_id}")
async def respond_to_approval(request_id: str, response: ApprovalResponse):
    """Respond to an approval request (approve or reject)."""
    logger.info(f"Processing {response.approver}'s response to request {request_id}")
    
    # Get client
    client = await get_client()
    
    # Check if the orchestration exists and is running
    status = client.get_orchestration_state(request_id)
    
    if not status:
        raise HTTPException(status_code=404, detail=f"Request {request_id} not found")
    
    # Check if the orchestration is in a state where we can respond to it
    if status.runtime_status not in [
        durable_client.OrchestrationStatus.RUNNING, 
        durable_client.OrchestrationStatus.PENDING
    ]:
        raise HTTPException(
            status_code=400, 
            detail=f"Cannot respond to request with status {status.runtime_status}"
        )
    
    # Send the approval response as an external event
    import datetime
    approval_data = {
        "is_approved": response.is_approved,
        "approver": response.approver,
        "comments": response.comments,
        "response_time": datetime.datetime.now().isoformat()  # Use current timestamp in ISO format
    }
    
    client.raise_orchestration_event(
        instance_id=request_id, 
        event_name="approval_response", 
        data=approval_data
    )
    
    approval_status = "approved" if response.is_approved else "rejected"
    return {
        "request_id": request_id,
        "message": f"Request {approval_status} by {response.approver}",
        "status_url": f"/api/requests/{request_id}"
    }

# Add root endpoint with instructions
@app.get("/")
async def root():
    """Root endpoint with API usage instructions"""
    return {
        "message": "Human Interaction Pattern API",
        "endpoints": {
            "create_request": "POST /api/requests",
            "check_status": "GET /api/requests/{request_id}",
            "respond_to_approval": "POST /api/approvals/{request_id}"
        },
        "documentation": "/docs"
    }

# Startup message
@app.on_event("startup")
async def startup_event():
    """Log startup information"""
    logger.info("Starting Human Interaction Pattern API")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
