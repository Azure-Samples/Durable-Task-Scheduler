import logging
import json
import time
import uuid
import azure.functions as func
import azure.durable_functions as df

# Create the Durable Functions app with HTTP auth level set to ANONYMOUS for easier testing
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

@app.route(route="async_http_api", methods=["POST"])
@app.durable_client_input(client_name="client")
async def http_start_async_operation(req: func.HttpRequest, client):
    """HTTP trigger that starts a long-running asynchronous operation."""
    try:
        # Get input from request body or use default
        req_body = req.get_json()
        operation_data = {
            "operation_type": req_body.get("operation_type", "data_processing") if req_body else "data_processing",
            "duration": req_body.get("duration", 30) if req_body else 30,  # seconds
            "data": req_body.get("data", {"sample": "data"}) if req_body else {"sample": "data"}
        }
        
        logging.info(f"Starting async operation: {operation_data}")
        
        # Start the orchestration
        instance_id = await client.start_new("async_http_api_orchestrator", client_input=operation_data)
        
        # Return management URLs for the orchestration
        return client.create_check_status_response(req, instance_id)
        
    except Exception as e:
        logging.error(f"Error starting orchestration: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to start orchestration: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )

@app.orchestration_trigger(context_name="context")
def async_http_api_orchestrator(context: df.DurableOrchestrationContext):
    """
    Orchestrator that demonstrates the async HTTP API pattern.
    Starts a long-running operation and provides status updates.
    """
    operation_data = context.get_input()
    logging.info(f"Async HTTP API orchestration started: {operation_data}")
    
    # Start the long-running operation
    operation_id = str(uuid.uuid4())
    operation_request = {
        **operation_data,
        "operation_id": operation_id,
        "started_at": context.current_utc_datetime.isoformat()
    }
    
    # Call the long-running activity
    result = yield context.call_activity("long_running_operation", operation_request)
    
    logging.info(f"Async HTTP API orchestration completed: {result}")
    return result

@app.activity_trigger(input_name="operationRequest")
def long_running_operation(operationRequest: dict) -> dict:
    """Simulate a long-running operation with status updates."""
    logging.info(f"Starting long-running operation: {operationRequest['operation_id']}")
    
    operation_type = operationRequest["operation_type"]
    duration = operationRequest["duration"]
    data = operationRequest["data"]
    
    # Simulate processing phases
    phases = ["initializing", "processing", "validating", "finalizing"]
    phase_duration = duration / len(phases)
    
    start_time = time.time()
    
    for i, phase in enumerate(phases):
        logging.info(f"Operation {operationRequest['operation_id']} - Phase: {phase}")
        
        # Simulate work being done in this phase
        time.sleep(phase_duration)
        
        progress = ((i + 1) / len(phases)) * 100
        logging.info(f"Operation {operationRequest['operation_id']} - Progress: {progress}%")
    
    end_time = time.time()
    actual_duration = end_time - start_time
    
    result = {
        "operation_id": operationRequest["operation_id"],
        "operation_type": operation_type,
        "status": "completed",
        "started_at": operationRequest["started_at"],
        "completed_at": time.strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
        "duration_seconds": round(actual_duration, 2),
        "input_data": data,
        "output_data": {
            "processed": True,
            "result_id": str(uuid.uuid4()),
            "phases_completed": phases,
            "total_items_processed": 1000,
            "success_rate": 98.5
        }
    }
    
    logging.info(f"Completed long-running operation: {operationRequest['operation_id']}")
    return result

@app.route(route="status/{instanceId}", methods=["GET"])
@app.durable_client_input(client_name="client")
async def get_orchestration_status(req: func.HttpRequest, client):
    """Get the status of a running orchestration with detailed progress information."""
    instance_id = req.route_params.get('instanceId')
    
    try:
        status = await client.get_status(instance_id)
        
        if status:
            # Enhanced status response for async operations
            response_data = {
                "instanceId": status.instance_id,
                "name": status.name,
                "runtimeStatus": status.runtime_status,
                "input": status.input_,
                "output": status.output,
                "createdTime": status.created_time.isoformat() if status.created_time else None,
                "lastUpdatedTime": status.last_updated_time.isoformat() if status.last_updated_time else None,
                "customStatus": status.custom_status
            }
            
            # Add operation-specific status information
            if status.runtime_status == "Running":
                response_data["message"] = "Operation is currently in progress. Check back later for results."
            elif status.runtime_status == "Completed":
                response_data["message"] = "Operation completed successfully."
            elif status.runtime_status == "Failed":
                response_data["message"] = "Operation failed. Check the output for error details."
            
            return func.HttpResponse(
                json.dumps(response_data, default=str),
                mimetype="application/json"
            )
        else:
            return func.HttpResponse(
                json.dumps({"error": "Orchestration not found"}),
                status_code=404,
                mimetype="application/json"
            )
            
    except Exception as e:
        logging.error(f"Error getting status: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to get status: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )

@app.route(route="cancel/{instanceId}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def cancel_operation(req: func.HttpRequest, client):
    """Cancel a running orchestration."""
    instance_id = req.route_params.get('instanceId')
    
    try:
        await client.terminate(instance_id, "Operation cancelled by user request")
        
        return func.HttpResponse(
            json.dumps({"message": f"Operation {instance_id} has been cancelled"}),
            mimetype="application/json"
        )
            
    except Exception as e:
        logging.error(f"Error cancelling operation: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to cancel operation: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )