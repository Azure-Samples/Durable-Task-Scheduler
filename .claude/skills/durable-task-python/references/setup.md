# Durable Task Scheduler Setup and Deployment

## Local Development with Emulator

### Docker Setup

```bash
# Pull the emulator image
docker pull mcr.microsoft.com/dts/dts-emulator:latest

# Run the emulator
docker run -d \
  -p 8080:8080 \
  -p 8082:8082 \
  --name dts-emulator \
  mcr.microsoft.com/dts/dts-emulator:latest

# Emulator endpoints:
# - gRPC: http://localhost:8080
# - Dashboard: http://localhost:8082
```

### Docker Compose

```yaml
version: '3.8'
services:
  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"  # gRPC endpoint
      - "8082:8082"  # Dashboard
    restart: unless-stopped
```

### Default Emulator Configuration

```python
# No authentication needed for local emulator
taskhub = "default"
endpoint = "http://localhost:8080"
credential = None
secure_channel = False
```

## Azure Durable Task Scheduler Provisioning

### Prerequisites

```bash
# Install Azure CLI
# https://learn.microsoft.com/cli/azure/install-azure-cli

# Login to Azure
az login

# Install durabletask extension
az extension add --name durabletask
```

### Create Scheduler and Task Hub

```bash
# Set variables
RESOURCE_GROUP="my-rg"
SCHEDULER_NAME="my-scheduler"
TASKHUB_NAME="my-taskhub"
LOCATION="eastus"

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create scheduler
az durabletask scheduler create \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --location $LOCATION \
  --ip-allowlist "[0.0.0.0/0]" \
  --sku-capacity 1 \
  --sku-name "Dedicated" \
  --tags "environment=dev"

# Create task hub
az durabletask taskhub create \
  --resource-group $RESOURCE_GROUP \
  --scheduler-name $SCHEDULER_NAME \
  --name $TASKHUB_NAME

# Get endpoint URL
az durabletask scheduler show \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --query "properties.endpoint" -o tsv
```

### Assign Permissions

```bash
# Get your user principal ID
USER_PRINCIPAL_ID=$(az ad signed-in-user show --query id -o tsv)

# Assign Durable Task Contributor role
az role assignment create \
  --assignee $USER_PRINCIPAL_ID \
  --role "Durable Task Contributor" \
  --scope "/subscriptions/{subscription-id}/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.DurableTask/schedulers/$SCHEDULER_NAME"
```

## Application Configuration

### Environment Variables

```bash
# Bash
export ENDPOINT="https://my-scheduler.region.durabletask.io"
export TASKHUB="my-taskhub"

# PowerShell
$env:ENDPOINT = "https://my-scheduler.region.durabletask.io"
$env:TASKHUB = "my-taskhub"
```

### Configuration Helper

```python
import os
from azure.identity import DefaultAzureCredential


def get_connection_config():
    """Get configuration for DTS connection"""
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    taskhub = os.getenv("TASKHUB", "default")
    
    is_local = endpoint == "http://localhost:8080"
    
    return {
        "host_address": endpoint,
        "taskhub": taskhub,
        "secure_channel": not is_local,
        "token_credential": None if is_local else DefaultAzureCredential()
    }
```

### Settings File Pattern

```python
# settings.py
import os
from dataclasses import dataclass
from azure.identity import DefaultAzureCredential


@dataclass
class DurableTaskSettings:
    endpoint: str
    taskhub: str
    secure_channel: bool
    credential: any
    
    @classmethod
    def from_environment(cls):
        endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
        taskhub = os.getenv("TASKHUB", "default")
        is_local = endpoint == "http://localhost:8080"
        
        return cls(
            endpoint=endpoint,
            taskhub=taskhub,
            secure_channel=not is_local,
            credential=None if is_local else DefaultAzureCredential()
        )


# Usage
settings = DurableTaskSettings.from_environment()
```

## Authentication Options

### Local Development (No Auth)

```python
credential = None
secure_channel = False
```

### DefaultAzureCredential (Recommended for Azure)

```python
from azure.identity import DefaultAzureCredential

credential = DefaultAzureCredential()
secure_channel = True
```

### Managed Identity

```python
from azure.identity import ManagedIdentityCredential

# System-assigned managed identity
credential = ManagedIdentityCredential()

# User-assigned managed identity
credential = ManagedIdentityCredential(client_id="<client-id>")
```

### Azure CLI Credential (Development)

```python
from azure.identity import AzureCliCredential

credential = AzureCliCredential()
```

## Worker Application Templates

### Console Worker

```python
#!/usr/bin/env python3
"""Durable Task Worker Application"""

import os
import signal
import sys
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker


# Activities
def my_activity(ctx: task.ActivityContext, input: str) -> str:
    return f"Processed: {input}"


# Orchestrations
def my_orchestration(ctx: task.OrchestrationContext, input: str):
    result = yield ctx.call_activity(my_activity, input=input)
    return result


def main():
    # Configuration
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    taskhub = os.getenv("TASKHUB", "default")
    is_local = endpoint == "http://localhost:8080"
    
    credential = None if is_local else DefaultAzureCredential()
    secure_channel = not is_local
    
    print(f"Starting worker...")
    print(f"  Endpoint: {endpoint}")
    print(f"  Task Hub: {taskhub}")
    
    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=secure_channel,
        taskhub=taskhub,
        token_credential=credential
    ) as worker:
        # Register orchestrations and activities
        worker.add_orchestrator(my_orchestration)
        worker.add_activity(my_activity)
        
        # Handle shutdown signals
        def shutdown(signum, frame):
            print("\nShutting down worker...")
            sys.exit(0)
        
        signal.signal(signal.SIGINT, shutdown)
        signal.signal(signal.SIGTERM, shutdown)
        
        # Start processing
        worker.start()
        
        print("Worker started. Press Ctrl+C to stop.")
        
        # Keep running
        while True:
            signal.pause()


if __name__ == "__main__":
    main()
```

### Flask/FastAPI Integration

```python
from flask import Flask, jsonify, request
from azure.identity import DefaultAzureCredential
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
import os

app = Flask(__name__)

# Initialize client
endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
taskhub = os.getenv("TASKHUB", "default")
is_local = endpoint == "http://localhost:8080"

client = DurableTaskSchedulerClient(
    host_address=endpoint,
    secure_channel=not is_local,
    taskhub=taskhub,
    token_credential=None if is_local else DefaultAzureCredential()
)


@app.route('/orchestrations', methods=['POST'])
def start_orchestration():
    """Start a new orchestration"""
    data = request.json
    instance_id = client.schedule_new_orchestration(
        "my_orchestration",
        input=data.get("input")
    )
    return jsonify({"instanceId": instance_id}), 202


@app.route('/orchestrations/<instance_id>', methods=['GET'])
def get_status(instance_id):
    """Get orchestration status"""
    state = client.get_orchestration_state(instance_id)
    if not state:
        return jsonify({"error": "Not found"}), 404
    
    return jsonify({
        "instanceId": instance_id,
        "status": state.runtime_status.name,
        "output": state.serialized_output
    })


@app.route('/orchestrations/<instance_id>/events/<event_name>', methods=['POST'])
def raise_event(instance_id, event_name):
    """Raise an event on an orchestration"""
    data = request.json
    client.raise_orchestration_event(instance_id, event_name, data=data)
    return jsonify({"status": "Event raised"}), 202
```

## Deployment Options

### Azure Container Apps

```yaml
# container-app.yaml
properties:
  configuration:
    secrets:
      - name: dts-endpoint
        value: "https://my-scheduler.region.durabletask.io"
    ingress:
      external: false
  template:
    containers:
      - image: myregistry.azurecr.io/durable-worker:latest
        name: worker
        env:
          - name: ENDPOINT
            secretRef: dts-endpoint
          - name: TASKHUB
            value: my-taskhub
        resources:
          cpu: 0.5
          memory: 1Gi
    scale:
      minReplicas: 1
      maxReplicas: 10
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: durable-worker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: durable-worker
  template:
    metadata:
      labels:
        app: durable-worker
    spec:
      containers:
        - name: worker
          image: myregistry.azurecr.io/durable-worker:latest
          env:
            - name: ENDPOINT
              valueFrom:
                secretKeyRef:
                  name: dts-config
                  key: endpoint
            - name: TASKHUB
              valueFrom:
                configMapKeyRef:
                  name: dts-config
                  key: taskhub
          resources:
            requests:
              cpu: 250m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
```

### Docker Image

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "worker.py"]
```

## Logging Configuration

```python
import logging

# Configure logging
log_handler = logging.FileHandler('durable.log', encoding='utf-8')
log_handler.setLevel(logging.DEBUG)
log_formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
log_handler.setFormatter(log_formatter)

# Apply to worker
with DurableTaskSchedulerWorker(
    host_address=endpoint,
    secure_channel=secure_channel,
    taskhub=taskhub,
    token_credential=credential,
    log_handler=log_handler,
    log_formatter=log_formatter
) as worker:
    # ...
```

## Monitoring

### Dashboard Access

- **Emulator**: http://localhost:8082
- **Azure**: Navigate to Scheduler → Task Hub → Dashboard URL in portal

### Query Orchestration Status

```python
# Get all running orchestrations
# (Note: SDK provides basic queries; use dashboard for advanced filtering)

# Check specific instance
state = client.get_orchestration_state(instance_id)
print(f"Status: {state.runtime_status}")
print(f"Created: {state.created_time}")
print(f"Updated: {state.last_updated_time}")
print(f"Input: {state.serialized_input}")
print(f"Output: {state.serialized_output}")
```
