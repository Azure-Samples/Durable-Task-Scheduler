# Function Chaining Pattern - Durable Functions with Durable Task Scheduler

This sample demonstrates the **Function Chaining** orchestration pattern using Durable Functions with the Durable Task Scheduler backend. In this pattern, activities are executed sequentially, with the output of each activity passed as input to the next activity.

## Pattern Overview

The Function Chaining pattern executes a sequence of activities in order:
1. `say_hello` - Takes a name and returns a greeting
2. `process_greeting` - Takes the greeting and adds more text
3. `finalize_response` - Takes the processed greeting and finalizes it

## Architecture

- **HTTP Trigger**: `function_chaining` - Starts the orchestration
- **Orchestrator**: `function_chaining_orchestrator` - Manages the sequence of activities
- **Activities**: Three sequential activities that transform the input
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

2. **Test the orchestration**:
   ```bash
   # Start orchestration with default input
   curl -X POST http://localhost:7071/api/function_chaining \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start orchestration with custom name
   curl -X POST http://localhost:7071/api/function_chaining \
     -H "Content-Type: application/json" \
     -d '{"name": "Alice"}'
   ```

3. **Check orchestration status**:
   Use the `statusQueryGetUri` from the response to check status, or:
   ```bash
   curl http://localhost:7071/api/status/{instanceId}
   ```

## Configuration Files

### host.json
Configures the Durable Functions extension to use Durable Task Scheduler:
- Sets the hub name to "default"
- Configures the storage provider as "AzureManaged"
- References the connection string name

### local.settings.json
Contains local development settings:
- Durable Task Scheduler connection string for local emulator
- Function worker runtime set to "python"

## Expected Output

For input `{"name": "Alice"}`, the orchestration will produce:
```
"Hello Alice! How are you today? I hope you're doing well!"
```

## Monitoring

- **Function Logs**: Check the Azure Functions host output for detailed logging
- **Dashboard**: Navigate to http://localhost:8082 to view orchestrations in the emulator dashboard

## Using with Azure Durable Task Scheduler

To use with an Azure-hosted Durable Task Scheduler instead of the emulator:

1. Update `local.settings.json`:
   ```json
   {
     "Values": {
       "DurableTaskSchedulerConnection": "Endpoint=https://your-scheduler.dts.azure.net;Authentication=DefaultAzure"
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
- [Function Chaining Pattern](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-sequence)