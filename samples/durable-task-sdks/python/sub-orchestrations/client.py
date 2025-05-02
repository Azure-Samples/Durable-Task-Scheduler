import os
from azure.identity import DefaultAzureCredential
from durabletask import client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Get environment variables for taskhub and endpoint with defaults
taskhub_name = os.getenv("TASKHUB", "default")
endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

print(f"Using taskhub: {taskhub_name}")
print(f"Using endpoint: {endpoint}")

# Set credential to None for emulator, or DefaultAzureCredential for Azure
credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

# Create a client, start an orchestration, and wait for it to finish
c = DurableTaskSchedulerClient(host_address=endpoint, 
                              secure_channel=endpoint != "http://localhost:8080",
                              taskhub=taskhub_name, 
                              token_credential=credential)

instance_id = c.schedule_new_orchestration("orchestrator")

state = c.wait_for_orchestration_completion(instance_id, timeout=30)

if state and state.runtime_status == client.OrchestrationStatus.COMPLETED:
    print(f'Orchestration completed! Result: {state.serialized_output}')
elif state:
    print(f'Orchestration failed: {state.failure_details}')
exit()
