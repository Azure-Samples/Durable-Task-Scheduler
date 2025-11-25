import logging
import json
import time
import uuid
from datetime import datetime, timedelta
import azure.functions as func
import azure.durable_functions as df

# Create the Durable Functions app with HTTP auth level set to ANONYMOUS for easier testing
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

@app.route(route="start_monitoring_job", methods=["POST"])
@app.durable_client_input(client_name="client")
async def http_start_monitoring_job(req: func.HttpRequest, client):
    """HTTP trigger that starts a job monitoring orchestration."""
    try:
        # Get input from request body or use default
        try:
            req_body = req.get_json() if req.get_body() else None
        except ValueError:
            req_body = None
            
        # Generate a unique job ID or use provided one
        job_id = req_body.get("job_id", f"job-{uuid.uuid4()}") if req_body else f"job-{uuid.uuid4()}"
        
        job_data = {
            "job_id": job_id,
            "polling_interval_seconds": req_body.get("polling_interval_seconds", 5) if req_body else 5,
            "timeout_seconds": req_body.get("timeout_seconds", 30) if req_body else 30
        }
        
        logging.info(f"Starting job monitoring for: {job_id}")
        logging.info(f"Polling interval: {job_data['polling_interval_seconds']} seconds")
        logging.info(f"Timeout: {job_data['timeout_seconds']} seconds")
        
        # Start the orchestration
        instance_id = await client.start_new("monitoring_job_orchestrator", client_input=job_data)
        
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
def monitoring_job_orchestrator(context: df.DurableOrchestrationContext):
    """
    Orchestrator that demonstrates the monitoring pattern.
    
    This orchestrator periodically checks the status of a job until it
    completes or reaches a maximum number of checks.
    """
    job_data = context.get_input()
    job_id = job_data.get("job_id")
    polling_interval = job_data.get("polling_interval_seconds", 5)
    timeout = job_data.get("timeout_seconds", 30)
    
    logging.info(f"Starting monitoring orchestration for job {job_id}")
    logging.info(f"Polling interval: {polling_interval} seconds")
    logging.info(f"Timeout: {timeout} seconds")
    
    # Record the start time
    start_time = context.current_utc_datetime
    expiration_time = start_time + timedelta(seconds=timeout)
    
    # Initialize monitoring state
    job_status = {
        "job_id": job_id,
        "status": "Unknown",
        "check_count": 0
    }
    
    # Loop until the job completes or times out
    while True:
        # Check current job status
        check_input = {"job_id": job_id, "check_count": job_status.get("check_count", 0)}
        job_status = yield context.call_activity("check_job_status", check_input)
        
        # Make the job status available to clients via custom status
        context.set_custom_status(job_status)
        
        if job_status["status"] == "Completed":
            logging.info(f"Job {job_id} completed after {job_status['check_count']} checks")
            break
        
        # Check if we've hit the timeout
        current_time = context.current_utc_datetime
        if current_time >= expiration_time:
            logging.info(f"Monitoring for job {job_id} timed out after {timeout} seconds")
            job_status["status"] = "Timeout"
            break
        
        # Wait using an activity function instead of timer (avoids timer configuration issues)
        logging.info(f"Waiting {polling_interval} seconds before next check of job {job_id}")
        yield context.call_activity("wait_for_interval", polling_interval)
    
    # Return the final status
    return {
        "job_id": job_id,
        "final_status": job_status["status"],
        "checks_performed": job_status["check_count"],
        "monitoring_duration_seconds": (context.current_utc_datetime - start_time).total_seconds()
    }

@app.activity_trigger(input_name="jobData")
def check_job_status(jobData: dict) -> dict:
    """
    Activity that simulates checking the status of a long-running job.
    In a real application, this would call an external API or service.
    """
    # Extract job_id from the job_data dictionary
    job_id = jobData.get("job_id", "unknown")
    check_count = jobData.get("check_count", 0)
    
    logging.info(f"Checking status for job: {job_id} (check #{check_count+1})")
    
    # Simulate job status progression
    if check_count >= 3:
        status = "Completed"
    else:
        status = "Running"
    
    return {
        "job_id": job_id,
        "status": status,
        "check_count": check_count + 1,
        "last_check_time": datetime.utcnow().isoformat()
    }

@app.activity_trigger(input_name="intervalSeconds")
def wait_for_interval(intervalSeconds: int) -> str:
    """
    Activity that simulates waiting for a specified interval.
    This is used instead of create_timer() to avoid timer configuration issues in Python SDK.
    """
    logging.info(f"Waiting for {intervalSeconds} seconds...")
    time.sleep(intervalSeconds)
    return f"Waited for {intervalSeconds} seconds"



@app.route(route="job_status/{jobId}", methods=["GET"])
@app.durable_client_input(client_name="client")
async def get_job_status(req: func.HttpRequest, client):
    """Get the current status of a job being monitored."""
    job_id = req.route_params.get('jobId')
    
    try:
        # In a real implementation, you would:
        # 1. Find the orchestration instance for this job_id
        # 2. Get its current status
        # For demo purposes, we'll return a simulated status
        
        # Simulate job status lookup
        status_info = {
            "job_id": job_id,
            "status": "Running",
            "progress_percent": 75,
            "estimated_completion": "2025-09-19T18:15:00Z",
            "last_updated": datetime.utcnow().isoformat(),
            "details": "Processing batch 3 of 4"
        }
        
        return func.HttpResponse(
            json.dumps(status_info),
            mimetype="application/json"
        )
        
    except Exception as e:
        logging.error(f"Error getting job status: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to get job status: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )