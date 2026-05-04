import asyncio
import logging
import uuid
import os
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
from azure.identity.aio import DefaultAzureCredential
from durabletask import client as durable_client
from durabletask.azuremanaged.client import AsyncDurableTaskSchedulerClient

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Models for request and response
class OperationRequest(BaseModel):
    processing_time: int = 5  # Default processing time in seconds

class OperationResponse(BaseModel):
    operation_id: str
    status_url: str

# Get environment variables for taskhub and endpoint with defaults
TASKHUB = os.getenv("TASKHUB", "default")
ENDPOINT = os.getenv("ENDPOINT", "http://localhost:8080")

print(f"Using taskhub: {TASKHUB}")
print(f"Using endpoint: {ENDPOINT}")

# Shared async client instance (managed by the app lifespan)
_async_client: AsyncDurableTaskSchedulerClient | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage the async client lifecycle with the FastAPI app."""
    global _async_client
    credential = None if ENDPOINT == "http://localhost:8080" else DefaultAzureCredential()
    _async_client = AsyncDurableTaskSchedulerClient(
        host_address=ENDPOINT,
        secure_channel=ENDPOINT != "http://localhost:8080",
        taskhub=TASKHUB,
        token_credential=credential,
    )
    yield
    await _async_client.close()
    _async_client = None


# Set up FastAPI app with lifespan
app = FastAPI(title="Durable Task Async HTTP API Sample", lifespan=lifespan)


async def get_client() -> AsyncDurableTaskSchedulerClient:
    """Get the async Durable Task client."""
    assert _async_client is not None, "Client not initialized — app not started"
    return _async_client

@app.post("/api/start-operation", response_model=OperationResponse)
async def start_operation(request: OperationRequest):
    """
    Start a long-running operation asynchronously.
    Returns an operation ID that can be used to check the status.
    """
    # Generate a unique operation ID
    operation_id = str(uuid.uuid4())
    logger.info(f"Starting new operation with ID: {operation_id}")
    
    # Get client
    client = await get_client()
    
    # Prepare input for the orchestration
    input_data = {
        "operation_id": operation_id,
        "processing_time": request.processing_time
    }
    
    # Schedule the orchestration using the async client
    instance_id = await client.schedule_new_orchestration(
        "async_http_api_orchestrator", 
        input=input_data,
        instance_id=operation_id  # Use operation_id as instance_id for simplicity
    )
    
    # Generate status URL for checking the result later
    status_url = f"/api/operations/{operation_id}"
    
    return OperationResponse(
        operation_id=operation_id,
        status_url=status_url
    )

@app.get("/api/operations/{operation_id}")
async def get_operation_status(operation_id: str):
    """
    Check the status of a previously started operation.
    Returns the operation result if it's complete, or status information if still running.
    """
    logger.info(f"Checking status for operation: {operation_id}")
    
    # Get client
    client = await get_client()
    
    # Get the orchestration status using the async client
    status = await client.get_orchestration_state(operation_id)
    
    if not status:
        raise HTTPException(status_code=404, detail=f"Operation {operation_id} not found")
    
    if status.runtime_status == durable_client.OrchestrationStatus.COMPLETED:
        # We need to parse the serialized_output if it exists
        import json
        result = None
        if hasattr(status, 'serialized_output') and status.serialized_output:
            try:
                result = json.loads(status.serialized_output)
            except json.JSONDecodeError:
                result = status.serialized_output
                
        return {
            "operation_id": operation_id,
            "status": "Completed",
            "result": result
        }
    elif status.runtime_status == durable_client.OrchestrationStatus.FAILED:
        return {
            "operation_id": operation_id,
            "status": "Failed",
            "error": str(status.failure_details)
        }
    else:
        # Still running
        # Use last_updated_at instead of last_updated_time (which doesn't exist)
        last_updated = None
        if hasattr(status, 'last_updated_at'):
            last_updated = status.last_updated_at
        
        return {
            "operation_id": operation_id,
            "status": str(status.runtime_status),
            "last_updated": last_updated
        }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
