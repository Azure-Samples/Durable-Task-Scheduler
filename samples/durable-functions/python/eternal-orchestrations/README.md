# Eternal Orchestrations Pattern - Durable Functions with Durable Task Scheduler

This sample demonstrates the **Eternal Orchestrations** pattern using Durable Functions with the Durable Task Scheduler backend. This pattern shows how to create long-running, recurring workflows that continue indefinitely until explicitly terminated.

## Pattern Overview

The Eternal Orchestrations pattern creates perpetual workflows:
1. **Initialization**: Sets up the recurring workflow with initial parameters
2. **Work Execution**: `perform_periodic_work` - Executes the recurring task
3. **Timer Wait**: Waits for the next execution interval using durable timers
4. **Self-Continuation**: Orchestration continues itself in a new iteration
5. **Graceful Termination**: Can be stopped via external events or conditions

## Architecture

- **HTTP Trigger**: `start_eternal_process` - Starts the eternal orchestration
- **Orchestrator**: `eternal_orchestrator` - Manages the infinite loop with timers
- **Activity**: `perform_periodic_work` - Executes the recurring work
- **Control Endpoints**: 
  - `stop_eternal/{instanceId}` - Gracefully stops the orchestration
  - `status/{instanceId}` - Checks current status and iteration count
- **Backend**: Durable Task Scheduler for long-term state management

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

2. **Start an eternal orchestration**:
   ```bash
   # Start with default parameters (2 minute intervals, 5 iterations max)
   curl -X POST http://localhost:7071/api/eternal_orchestration \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start with custom parameters
   curl -X POST http://localhost:7071/api/eternal_orchestration \
     -H "Content-Type: application/json" \
     -d '{"task_type": "data_sync", "interval_minutes": 1, "max_iterations": 10, "target_url": "https://httpbin.org/delay/1"}'
   ```

3. **Monitor the running orchestration**:
   ```bash
   # Check current status and iteration count
   curl http://localhost:7071/api/status/{instanceId}
   ```

4. **Stop the eternal orchestration**:
   ```bash
   # Send stop signal
   curl -X POST http://localhost:7071/api/stop/{instanceId}
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

### Running Status
```json
{
  "runtimeStatus": "Running",
  "input": {"task_type": "health_check", "interval_minutes": 2, "max_iterations": 5},
  "customStatus": {
    "current_iteration": 3,
    "max_iterations": 5,
    "last_run": "2025-09-19T16:45:30Z",
    "task_type": "health_check"
  }
}
```

### Completed Status (after reaching max iterations)
```json
{
  "runtimeStatus": "Completed", 
  "output": {
    "status": "completed",
    "total_iterations": 5,
    "final_result": {
      "iteration": 5,
      "task_type": "health_check",
      "success": true,
      "executed_at": "2025-09-19T16:45:30Z"
    }
  }
}
```

## How It Works

1. **Continuous Loop**: Orchestration runs in an infinite loop until stopped
2. **Durable Timers**: Uses `create_timer()` to wait between iterations without consuming resources
3. **Self-Continuation**: Each iteration calls `continue_as_new()` to reset and continue
4. **External Events**: Listens for stop signals during timer waits
5. **State Preservation**: Maintains iteration counters and status across continuations

## Key Features

- **Resource Efficient**: Uses durable timers instead of blocking waits
- **Long-Term Stability**: Designed to run for days, weeks, or months
- **Graceful Shutdown**: Responds to external stop events
- **Progress Tracking**: Maintains iteration counts and execution history
- **Self-Healing**: Automatically recovers from host restarts

## Monitoring

- **Function Logs**: Check the Azure Functions host output for iteration details
- **Dashboard**: Navigate to http://localhost:8082 to view long-running orchestrations
- **Custom Status**: Monitor iteration progress and timing via status endpoints
- **Timers**: See active timers and their scheduled execution times

## Best Practices

- **Set Reasonable Intervals**: Avoid very short intervals that could overwhelm the system
- **Implement Stop Conditions**: Always provide a way to gracefully terminate
- **Monitor Resource Usage**: Watch for memory leaks or resource accumulation
- **Handle Failures**: Include error handling and retry logic in activities

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
- [Eternal Orchestrations Pattern](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)