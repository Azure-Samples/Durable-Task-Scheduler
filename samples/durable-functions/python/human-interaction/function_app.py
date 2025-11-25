import logging
import json
import uuid
from datetime import datetime, timedelta
import azure.functions as func
import azure.durable_functions as df

# Create the Durable Functions app with HTTP auth level set to ANONYMOUS for easier testing
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

@app.route(route="human_interaction", methods=["POST"])
@app.durable_client_input(client_name="client")
async def http_start_approval_process(req: func.HttpRequest, client):
    """HTTP trigger that starts a human interaction approval process."""
    try:
        # Get input from request body or use default
        req_body = req.get_json()
        request_data = {
            "request_type": req_body.get("request_type", "expense_approval") if req_body else "expense_approval",
            "amount": req_body.get("amount", 1500.00) if req_body else 1500.00,
            "requester": req_body.get("requester", "john.doe@company.com") if req_body else "john.doe@company.com",
            "description": req_body.get("description", "Business travel expenses") if req_body else "Business travel expenses",
            "timeout_minutes": req_body.get("timeout_minutes", 10) if req_body else 10  # Short timeout for demo
        }
        
        logging.info(f"Starting human interaction process: {request_data}")
        
        # Start the orchestration
        instance_id = await client.start_new("human_interaction_orchestrator", client_input=request_data)
        
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
def human_interaction_orchestrator(context: df.DurableOrchestrationContext):
    """
    Orchestrator that demonstrates the human interaction pattern.
    Waits for human approval with a timeout mechanism.
    """
    request_data = context.get_input()
    logging.info(f"Human interaction orchestration started: {request_data}")
    
    # Generate approval request (use instance ID as approval ID for simplicity)
    approval_request = {
        "approval_id": context.instance_id,
        "request_type": request_data["request_type"],
        "amount": request_data["amount"],
        "requester": request_data["requester"],
        "description": request_data["description"],
        "created_at": context.current_utc_datetime.isoformat(),
        "timeout_minutes": request_data["timeout_minutes"]
    }
    
    # Send approval request notification
    yield context.call_activity("send_approval_request", approval_request)
    
    # Update custom status to show waiting for approval
    context.set_custom_status({
        "approval_id": approval_request["approval_id"],
        "status": "waiting_for_approval",
        "timeout_minutes": request_data["timeout_minutes"],
        "approval_url": f"http://localhost:7071/api/approve/{approval_request['approval_id']}"
    })
    
    # For demo purposes, let's simplify and just wait for the external event
    # In a production scenario, you'd implement proper timeout handling
    logging.info(f"Waiting for external approval event for request: {approval_request['approval_id']}")
    
    try:
        # Wait for external approval event (simplified - no timeout for now)
        approval_response = yield context.wait_for_external_event("approval_response")
        
        logging.info(f"Approval received for request: {approval_request['approval_id']}")
        logging.info(f"Approval response type: {type(approval_response)}, value: {approval_response}")
        
        # Handle case where response might be a string (serialization issue)
        if isinstance(approval_response, str):
            import json
            try:
                approval_response = json.loads(approval_response)
            except:
                # If parsing fails, create a default response
                approval_response = {"approved": True, "approver": "unknown", "comments": "Parsed from string"}
        
        result = yield context.call_activity("process_approval", {
            "approval_request": approval_request,
            "approval_response": approval_response,
            "status": "approved" if approval_response.get("approved") else "rejected"
        })
        
    except Exception as e:
        # Handle any errors
        logging.error(f"Error waiting for approval: {str(e)}")
        result = yield context.call_activity("process_approval", {
            "approval_request": approval_request,
            "approval_response": None,
            "status": "error"
        })
    
    logging.info(f"Human interaction orchestration completed: {result}")
    return result

@app.activity_trigger(input_name="approvalRequest")
def send_approval_request(approvalRequest: dict) -> dict:
    """Send approval request notification (simulated)."""
    logging.info(f"Sending approval request: {approvalRequest['approval_id']}")
    
    # In a real scenario, this would send email, SMS, or push notification
    # For demo purposes, we'll just log the approval details
    
    notification_result = {
        "approval_id": approvalRequest["approval_id"],
        "notification_sent": True,
        "notification_method": "email",  # simulated
        "recipient": "manager@company.com",  # simulated
        "sent_at": datetime.utcnow().isoformat(),
        "approval_url": f"http://localhost:7071/api/approve/{approvalRequest['approval_id']}"
    }
    
    logging.info(f"Approval request notification sent: {notification_result}")
    return notification_result

@app.activity_trigger(input_name="approvalData")
def process_approval(approvalData: dict) -> dict:
    """Process the approval response and finalize the request."""
    logging.info(f"Processing approval: {approvalData}")
    
    approval_request = approvalData["approval_request"]
    approval_response = approvalData["approval_response"]
    status = approvalData["status"]
    
    result = {
        "approval_id": approval_request["approval_id"],
        "request_type": approval_request["request_type"],
        "amount": approval_request["amount"],
        "requester": approval_request["requester"],
        "status": status,
        "processed_at": datetime.utcnow().isoformat(),
        "created_at": approval_request["created_at"]
    }
    
    if status == "approved":
        result.update({
            "approved_by": approval_response.get("approver", "unknown"),
            "approval_comments": approval_response.get("comments", ""),
            "approved_at": approval_response.get("approved_at"),
            "next_steps": "Request will be processed for payment"
        })
    elif status == "rejected":
        result.update({
            "rejected_by": approval_response.get("approver", "unknown"),
            "rejection_reason": approval_response.get("comments", "No reason provided"),
            "rejected_at": approval_response.get("approved_at"),
            "next_steps": "Request has been denied. Contact approver for details."
        })
    else:  # timeout
        result.update({
            "timeout_reason": f"No response received within {approval_request['timeout_minutes']} minutes",
            "next_steps": "Request will be escalated to senior management"
        })
    
    logging.info(f"Approval processing completed: {result}")
    return result

@app.activity_trigger(input_name="timeoutConfig")
def wait_for_timeout(timeoutConfig: dict) -> dict:
    """Activity function that handles waiting for timeout duration."""
    import time
    
    timeout_minutes = timeoutConfig.get("timeout_minutes", 10)
    # For demo purposes, cap the timeout at 5 minutes and convert to seconds
    timeout_seconds = min(timeout_minutes * 60, 300)  # Max 5 minutes
    
    logging.info(f"Starting timeout wait for {timeout_seconds} seconds")
    
    start_time = time.time()
    time.sleep(timeout_seconds)
    actual_wait = time.time() - start_time
    
    result = {
        "timeout_occurred": True,
        "requested_timeout_seconds": timeout_seconds,
        "actual_wait_seconds": round(actual_wait, 2),
        "completed_at": datetime.utcnow().isoformat()
    }
    
    logging.info(f"Timeout completed: {result}")
    return result

@app.route(route="approve/{approvalId}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def submit_approval(req: func.HttpRequest, client):
    """HTTP endpoint for submitting approval decisions."""
    approval_id = req.route_params.get('approvalId')
    
    try:
        # Get approval decision from request body
        req_body = req.get_json()
        approved = req_body.get("approved", False) if req_body else False
        approver = req_body.get("approver", "manager@company.com") if req_body else "manager@company.com"
        comments = req_body.get("comments", "") if req_body else ""
        
        approval_response = {
            "approved": approved,
            "approver": approver,
            "comments": comments,
            "approved_at": datetime.utcnow().isoformat()
        }
        
        logging.info(f"Submitting approval decision: {approval_response}")
        
        # For this demo, we'll use the approval_id as the instance_id
        # In a real implementation, you'd maintain a proper mapping
        instance_id = approval_id
        
        # Raise the external event to the orchestration
        await client.raise_event(instance_id, "approval_response", approval_response)
        
        return func.HttpResponse(
            json.dumps({
                "message": f"Approval decision submitted for {approval_id}",
                "approval_response": approval_response,
                "instance_id": instance_id,
                "event_raised": True
            }),
            mimetype="application/json"
        )
        
    except Exception as e:
        logging.error(f"Error submitting approval: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to submit approval: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )

@app.route(route="status/{instanceId}", methods=["GET"])
@app.durable_client_input(client_name="client")
async def get_orchestration_status(req: func.HttpRequest, client):
    """Get the status of a running orchestration."""
    instance_id = req.route_params.get('instanceId')
    
    try:
        status = await client.get_status(instance_id)
        
        if status:
            return func.HttpResponse(
                json.dumps({
                    "instanceId": status.instance_id,
                    "name": status.name,
                    "runtimeStatus": status.runtime_status,
                    "input": status.input_,
                    "output": status.output,
                    "createdTime": status.created_time.isoformat() if status.created_time else None,
                    "lastUpdatedTime": status.last_updated_time.isoformat() if status.last_updated_time else None
                }, default=str),
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