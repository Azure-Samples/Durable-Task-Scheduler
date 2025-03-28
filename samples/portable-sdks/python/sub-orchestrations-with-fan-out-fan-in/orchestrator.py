import os
from azure.identity import DefaultAzureCredential
from durabletask import client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

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

# Create a client, start an orchestration, and wait for it to finish
c = DurableTaskSchedulerClient(host_address=endpoint, secure_channel=True,
                               taskhub=taskhub_name, token_credential=credential)

instance_id = c.schedule_new_orchestration("orchestrator")

state = c.wait_for_orchestration_completion(instance_id, timeout=30)

if state and state.runtime_status == client.OrchestrationStatus.COMPLETED:
    print(f'Orchestration completed! Result: {state.serialized_output}')
elif state:
    print(f'Orchestration failed: {state.failure_details}')
exit()
