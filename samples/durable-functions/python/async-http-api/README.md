# Async HTTP API Pattern - Durable Functions with Durable Task Scheduler

This sample demonstrates the **Async HTTP API** orchestration pattern using Durable Functions with the Durable Task Scheduler backend. This pattern shows how to handle long-running operations with HTTP polling for status updates.

## Pattern Overview

The Async HTTP API pattern provides a way to handle long-running operations:
1. **HTTP Start**: Client submits a request and receives URLs for status checking
2. **Long-Running Process**: `process_long_running_task` simulates work that takes time
3. **Status Polling**: Clients can check progress via HTTP endpoints
4. **Completion**: Eventually returns the final result

## Architecture

- **HTTP Trigger**: `async_operation` - Starts the long-running orchestration
- **Orchestrator**: `async_http_orchestrator` - Manages the long-running process
- **Activity**: `process_long_running_task` - Simulates time-consuming work
- **Status Endpoint**: `status/{instanceId}` - Provides orchestration status
- **Backend**: Durable Task Scheduler for state management

## Prerequisites

- [Python 3.9+](https://www.python.org/downloads/)
- [Azure Functions Core Tools](https://docs.microsoft.com/azure/azure-functions/functions-run-local#install-the-azure-functions-core-tools)
- [Durable Task Scheduler Emulator](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler) (for local development)

## Setup

1. **Install dependencies**:
   ```bash
   pip install -r requirements.txt
   ```

2. **Start the Durable Task Scheduler Emulator**:
   ```bash
   docker run --name dts-emulator -p 8080:8080 -p 8082:8082 -d mcr.microsoft.com/dts/dts-emulator:latest
   ```

3. **Configure connection** (already set in `local.settings.json`):
   The sample is configured to use the local emulator by default.

## Running the Sample

1. **Start the Azure Functions host**:
   ```bash
   func start
   ```

2. **Start a long-running operation**:
   ```bash
   # Start operation with default parameters
   curl -X POST http://localhost:7071/api/async_http_api \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start operation with custom parameters
   curl -X POST http://localhost:7071/api/async_http_api \
     -H "Content-Type: application/json" \
     -d '{"operation_type": "data_processing", "duration": 45, "data": {"input": "sample"}}'
   ```

3. **Poll for status** (use URLs from the initial response):
   ```bash
   # Check status
   curl http://localhost:7071/api/status/{instanceId}
   
   # The response includes standard orchestration management URLs:
   # - statusQueryGetUri: Check current status
   # - sendEventPostUri: Send external events
   # - terminatePostUri: Terminate the orchestration
   ```

## Configuration Files

### host.json
Configures the Durable Functions extension to use Durable Task Scheduler:
- Sets the hub name to "default"
- Configures the storage provider as "azureManaged" 
- References the connection string name

### local.settings.json
Contains local development settings:
- Durable Task Scheduler connection string for local emulator
- Function worker runtime set to "python"

## Expected Behavior

1. **Initial Response**: Returns HTTP 202 with management URLs
2. **Status Polling**: Shows "Running" status while processing
3. **Progress Updates**: Activity logs progress during execution
4. **Completion**: Eventually returns "Completed" with final result

Example status progression:
```json
// Initially
{"runtimeStatus": "Running", "output": null}

// Finally  
{"runtimeStatus": "Completed", "output": {"task": "ProcessData", "result": "Success", "duration": 30}}
```

## How It Works

1. **Async Start**: HTTP trigger starts orchestration and returns immediately with status URLs
2. **Background Processing**: Long-running activity executes while client can poll for updates
3. **Status Management**: Durable Functions manages orchestration state across the operation
4. **Client Experience**: Clients get immediate response and can check progress periodically

## Monitoring

- **Function Logs**: Check the Azure Functions host output for processing details
- **Dashboard**: Navigate to http://localhost:8082 to view long-running orchestrations
- **Status Endpoints**: Use the returned URLs to monitor progress programmatically

## Using with Azure Durable Task Scheduler

To use with an Azure-hosted Durable Task Scheduler instead of the emulator:

1. Update `local.settings.json`:
   ```json
   {
     "Values": {
       "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=https://your-scheduler.dts.azure.net;Authentication=DefaultAzure"
     }
   }
   ```

2. Ensure you're authenticated with Azure CLI:
   ```bash
   az login
   ```

## Learn More

- [Durable Functions Overview](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
- [Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Async HTTP APIs Pattern](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-http-features)