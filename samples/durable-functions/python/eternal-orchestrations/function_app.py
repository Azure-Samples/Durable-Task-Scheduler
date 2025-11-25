import logging
import json
from datetime import datetime, timedelta
import azure.functions as func
import azure.durable_functions as df

# Create the Durable Functions app with HTTP auth level set to ANONYMOUS for easier testing
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

@app.route(route="eternal_orchestration", methods=["POST"])
@app.durable_client_input(client_name="client")
async def http_start_eternal_orchestration(req: func.HttpRequest, client):
    """HTTP trigger that starts an eternal orchestration."""
    try:
        # Get input from request body or use default
        req_body = req.get_json()
        config = {
            "task_type": req_body.get("task_type", "health_check") if req_body else "health_check",
            "interval_minutes": req_body.get("interval_minutes", 2) if req_body else 2,  # Short interval for demo
            "max_iterations": req_body.get("max_iterations", 5) if req_body else 5,  # Limit for demo
            "target_url": req_body.get("target_url", "https://httpbin.org/status/200") if req_body else "https://httpbin.org/status/200"
        }
        
        logging.info(f"Starting eternal orchestration: {config}")
        
        # Start the orchestration
        instance_id = await client.start_new("eternal_orchestrator", client_input=config)
        
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
def eternal_orchestrator(context: df.DurableOrchestrationContext):
    """
    Orchestrator that demonstrates the eternal orchestration pattern.
    Uses continue_as_new for proper eternal orchestration behavior.
    """
    config = context.get_input()
    current_iteration = config.get("current_iteration", 1)
    max_iterations = config.get("max_iterations", 5)
    
    logging.info(f"Eternal orchestration iteration {current_iteration}")
    
    # Update custom status
    context.set_custom_status({
        "current_iteration": current_iteration,
        "max_iterations": max_iterations,
        "last_run": context.current_utc_datetime.isoformat(),
        "task_type": config["task_type"],
        "status": "executing_task"
    })
    
    # Execute the periodic task
    task_result = yield context.call_activity("execute_periodic_task", {
        **config,
        "iteration": current_iteration
    })
    
    # Check if we should continue
    if current_iteration >= max_iterations:
        logging.info(f"Eternal orchestration completed after {current_iteration} iterations")
        return {
            "status": "completed",
            "total_iterations": current_iteration,
            "final_result": task_result
        }
    
    # Update status to waiting
    context.set_custom_status({
        "current_iteration": current_iteration,
        "max_iterations": max_iterations,
        "last_run": context.current_utc_datetime.isoformat(),
        "task_type": config["task_type"],
        "status": "waiting_for_next_iteration"
    })
    
    # Use activity function to handle the delay instead of orchestrator timer
    yield context.call_activity("wait_for_interval", {
        "interval_minutes": config.get("interval_minutes", 2)
    })
    
    # Continue with next iteration
    next_config = config.copy()
    next_config["current_iteration"] = current_iteration + 1
    context.continue_as_new(next_config)

@app.activity_trigger(input_name="taskConfig")
def execute_periodic_task(taskConfig: dict) -> dict:
    """Execute a periodic task (e.g., health check, data sync, cleanup)."""
    import requests
    import time
    
    task_type = taskConfig["task_type"]
    iteration = taskConfig["iteration"]
    target_url = taskConfig["target_url"]
    
    logging.info(f"Executing {task_type} task - iteration {iteration}")
    
    start_time = time.time()
    
    try:
        if task_type == "health_check":
            # Perform health check
            response = requests.get(target_url, timeout=10)
            success = response.status_code == 200
            details = {
                "status_code": response.status_code,
                "response_time_ms": round((time.time() - start_time) * 1000, 2),
                "url": target_url
            }
        elif task_type == "data_sync":
            # Simulate data synchronization
            time.sleep(1)  # Simulate work
            success = True
            details = {
                "records_synced": 150,
                "duration_ms": round((time.time() - start_time) * 1000, 2)
            }
        elif task_type == "cleanup":
            # Simulate cleanup task
            time.sleep(0.5)  # Simulate work
            success = True
            details = {
                "files_cleaned": 25,
                "space_freed_mb": 128,
                "duration_ms": round((time.time() - start_time) * 1000, 2)
            }
        else:
            # Default task
            success = True
            details = {"message": f"Executed {task_type} task"}
            
    except Exception as e:
        success = False
        details = {"error": str(e)}
    
    result = {
        "iteration": iteration,
        "task_type": task_type,
        "success": success,
        "executed_at": datetime.utcnow().isoformat(),
        "details": details
    }
    
    logging.info(f"Task execution completed: {result}")
    return result

@app.activity_trigger(input_name="waitConfig")
def wait_for_interval(waitConfig: dict) -> dict:
    """Activity function that handles waiting between iterations."""
    import time
    
    interval_minutes = waitConfig.get("interval_minutes", 2)
    # For demo purposes, cap the wait time at 2 minutes and convert to seconds
    wait_seconds = min(interval_minutes * 60, 120)
    
    logging.info(f"Waiting {wait_seconds} seconds before next iteration")
    
    start_time = time.time()
    time.sleep(wait_seconds)
    actual_wait = time.time() - start_time
    
    result = {
        "requested_wait_seconds": wait_seconds,
        "actual_wait_seconds": round(actual_wait, 2),
        "completed_at": datetime.utcnow().isoformat()
    }
    
    logging.info(f"Wait completed: {result}")
    return result

@app.route(route="stop/{instanceId}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def stop_eternal_orchestration(req: func.HttpRequest, client):
    """Stop an eternal orchestration."""
    instance_id = req.route_params.get('instanceId')
    
    try:
        await client.terminate(instance_id, "Stopped by user request")
        
        return func.HttpResponse(
            json.dumps({"message": f"Eternal orchestration {instance_id} has been stopped"}),
            mimetype="application/json"
        )
            
    except Exception as e:
        logging.error(f"Error stopping orchestration: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to stop orchestration: {str(e)}"}),
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
                    "customStatus": status.custom_status,
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