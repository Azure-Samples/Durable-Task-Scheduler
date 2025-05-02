import os
from datetime import timedelta
import time  

from azure.identity import DefaultAzureCredential
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

def cleanup_task(ctx, _) -> str:
    """Activity function that performs cleanup"""
    print('Performing cleanup...')

    # Simulate cleanup process
    time.sleep(5)

    return 'Cleanup completed'

def periodic_cleanup(ctx, counter):
    yield ctx.call_activity(cleanup_task)
    yield ctx.create_timer(timedelta(seconds=15))

    if counter < 5:
        ctx.continue_as_new(counter + 1)

# Get environment variables for taskhub and endpoint with defaults
taskhub_name = os.getenv("TASKHUB", "default")
endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

print(f"Using taskhub: {taskhub_name}")
print(f"Using endpoint: {endpoint}")
# Set credential to None for emulator, or DefaultAzureCredential for Azure
credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

# Configure and start the worker
with DurableTaskSchedulerWorker(host_address=endpoint, 
                               secure_channel=endpoint != "http://localhost:8080",
                               taskhub=taskhub_name, 
                               token_credential=credential) as w:
    
    w.add_orchestrator(periodic_cleanup)
    w.add_activity(cleanup_task)

    w.start()

    while True:
        time.sleep(1)