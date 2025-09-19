# Fan-Out/Fan-In Pattern - Durable Functions with Durable Task Scheduler

This sample demonstrates the **Fan-Out/Fan-In** orchestration pattern using Durable Functions with the Durable Task Scheduler backend. In this pattern, multiple activities are executed in parallel (fan-out), and their results are aggregated when all complete (fan-in).

## Pattern Overview

The Fan-Out/Fan-In pattern executes multiple activities in parallel and aggregates results:
1. **Fan-Out**: `process_work_item` activities are started in parallel for each work item
2. **Fan-In**: Wait for all parallel activities to complete
3. `aggregate_results` - Combines all results into a summary

## Architecture

- **HTTP Trigger**: `fan_out_fan_in` - Starts the orchestration
- **Orchestrator**: `fan_out_fan_in_orchestrator` - Manages parallel execution and aggregation
- **Activities**: 
  - `process_work_item` - Processes individual work items (executed in parallel)
  - `aggregate_results` - Combines results from all parallel activities
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
   # Start orchestration with default work items
   curl -X POST http://localhost:7071/api/fan_out_fan_in \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start orchestration with custom work items
   curl -X POST http://localhost:7071/api/fan_out_fan_in \
     -H "Content-Type: application/json" \
     -d '{"workItems": ["Task1", "Task2", "Task3", "Task4", "Task5", "Task6"]}'
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
- Configures the storage provider as "azureManaged"
- References the connection string name

### local.settings.json
Contains local development settings:
- Durable Task Scheduler connection string for local emulator
- Function worker runtime set to "python"

## Expected Output

For input `{"workItems": ["Task1", "Task2", "Task3"]}`, the orchestration will produce:
```json
{
  "total_items_processed": 3,
  "total_value": 165,
  "average_value": 55.0,
  "total_processing_time": 3.47,
  "processed_items": ["Processed_Task1", "Processed_Task2", "Processed_Task3"],
  "success": true
}
```

## How It Works

1. **Parallel Execution**: Each work item is processed simultaneously using `process_work_item`
2. **Processing Simulation**: Each activity simulates work with random processing times (0.5-2.0 seconds)
3. **Result Generation**: Each activity returns processing metrics and a random value
4. **Aggregation**: Results are combined to show totals, averages, and processing statistics

## Monitoring

- **Function Logs**: Check the Azure Functions host output for detailed logging of parallel execution
- **Dashboard**: Navigate to http://localhost:8082 to view orchestrations and parallel activities in the emulator dashboard
- **Performance**: Watch how multiple activities execute concurrently and complete at different times

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
- [Fan-Out/Fan-In Pattern](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-cloud-backup)