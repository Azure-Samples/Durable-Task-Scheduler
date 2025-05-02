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
def check_job_status(ctx, job_data: dict) -> dict:
    """
    Activity that simulates checking the status of a long-running job.
    In a real application, this would call an external API or service.
    """
    # Extract job_id from the job_data dictionary
    job_id = job_data.get("job_id", "unknown")
    check_count = job_data.get("check_count", 0)
    
    logger.info(f"Checking status for job: {job_id} (check #{check_count+1})")
    
    # Simulate job status
    if check_count >= 3:
        status = "Completed"
    else:
        status = "Running"
    
    return {
        "job_id": job_id,
        "status": status,
        "check_count": check_count + 1,
        "last_check_time": datetime.datetime.now().isoformat()
    }

# Orchestrator function
def monitoring_job_orchestrator(ctx, job_data: dict) -> dict:
    """
    Orchestrator that demonstrates the monitoring pattern.
    
    This orchestrator periodically checks the status of a job until it
    completes or reaches a maximum number of checks.
    """
    job_id = job_data.get("job_id")
    polling_interval = job_data.get("polling_interval_seconds", 5)
    timeout = job_data.get("timeout_seconds", 30)
    
    logger.info(f"Starting monitoring orchestration for job {job_id}")
    logger.info(f"Polling interval: {polling_interval} seconds")
    logger.info(f"Timeout: {timeout} seconds")
    
    # Record the start time
    start_time = ctx.current_utc_datetime
    expiration_time = start_time + datetime.timedelta(seconds=timeout)
    
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
        job_status = yield ctx.call_activity("check_job_status", input=check_input)
        
        # Make the job status available to clients via custom status
        ctx.set_custom_status(job_status)
        
        if job_status["status"] == "Completed":
            logger.info(f"Job {job_id} completed after {job_status['check_count']} checks")
            break
        
        # Check if we've hit the timeout
        current_time = ctx.current_utc_datetime
        if current_time >= expiration_time:
            logger.info(f"Monitoring for job {job_id} timed out after {timeout} seconds")
            job_status["status"] = "Timeout"
            break
        
        # Determine the next check time
        next_check_time = current_time + datetime.timedelta(seconds=polling_interval)
        
        # Don't check past the expiration time
        if next_check_time > expiration_time:
            next_check_time = expiration_time
        
        # Schedule the next check
        logger.info(f"Waiting {polling_interval} seconds before next check of job {job_id}")
        yield ctx.create_timer(next_check_time)
    
    # Return the final status
    return {
        "job_id": job_id,
        "final_status": job_status["status"],
        "checks_performed": job_status["check_count"],
        "monitoring_duration_seconds": (ctx.current_utc_datetime - start_time).total_seconds()
    }

async def main():
    """Main entry point for the worker process."""
    logger.info("Starting Monitoring pattern worker...")
    
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
        worker.add_activity(check_job_status)
        worker.add_orchestrator(monitoring_job_orchestrator)
        
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
