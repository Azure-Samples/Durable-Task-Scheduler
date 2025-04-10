import os
from azure.identity import DefaultAzureCredential
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Get environment variables for taskhub and endpoint with defaults
taskhub_name = os.getenv("TASKHUB", "default")
endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

print(f"Using taskhub: {taskhub_name}")
print(f"Using endpoint: {endpoint}")

# Set credential to None for emulator, or DefaultAzureCredential for Azure
credential = None if endpoint == "http://localhost:8080" else DefaultAzureCredential()

# Create a client, start an orchestration, and wait for it to finish
c = DurableTaskSchedulerClient(host_address=endpoint, secure_channel=True,
                               taskhub=taskhub_name, token_credential=credential)

counter = 1
instance_id = c.schedule_new_orchestration("periodic_cleanup", input=counter)

