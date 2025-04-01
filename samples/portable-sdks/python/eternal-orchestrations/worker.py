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

# Read the environment variable
taskhub_name = os.getenv("TASKHUB")

# Check if the variable exists
if taskhub_name:
    print(f"The value of TASKHUB is: {taskhub_name}")
else:
    print("TASKHUB is not set. Please set the TASKHUB environment variable to the name of the taskhub you wish to use")
    print("If you are using windows powershell, run the following: $env:TASKHUB=\"<taskhubname>\"")
    print("If you are using bash, run the following: export TASKHUB=\"<taskhubname>\"")
    exit()

# Read the environment variable
endpoint = os.getenv("ENDPOINT")

# Check if the variable exists
if endpoint:
    print(f"The value of ENDPOINT is: {endpoint}")
else:
    print("ENDPOINT is not set. Please set the ENDPOINT environment variable to the endpoint of the scheduler")
    print("If you are using windows powershell, run the following: $env:ENDPOINT=\"<schedulerEndpoint>\"")
    print("If you are using bash, run the following: export ENDPOINT=\"<schedulerEndpoint>\"")
    exit()

credential = DefaultAzureCredential()

# Configure and start the worker
with DurableTaskSchedulerWorker(host_address=endpoint, secure_channel=True,
                                taskhub=taskhub_name, token_credential=credential) as w:
    
    w.add_orchestrator(periodic_cleanup)
    w.add_activity(cleanup_task)

    w.start()

    while True:
        time.sleep(1)