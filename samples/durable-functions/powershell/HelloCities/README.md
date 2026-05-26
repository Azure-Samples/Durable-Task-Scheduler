# Hello Cities — Durable Functions PowerShell Quickstart

PowerShell | Durable Functions

## Description

This quickstart demonstrates Durable Functions with PowerShell using the Durable Task Scheduler backend. It includes two patterns:

1. **Function Chaining** — An orchestration that calls three "say hello" activities sequentially
2. **Fan-out/Fan-in** — An orchestration that greets multiple cities in parallel and aggregates results

## Prerequisites

1. [PowerShell 7.4+](https://learn.microsoft.com/powershell/scripting/install/installing-powershell)
2. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

## Quick Run

1. Start the emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Navigate to the sample directory:
   ```bash
   cd samples/durable-functions/powershell/HelloCities
   ```

3. The included `local.settings.json` is pre-configured to connect to the emulator. Run the function app:
   ```bash
   func start
   ```

4. Trigger the function chaining orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartChaining
   ```

5. Trigger the fan-out/fan-in orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartFanOutFanIn
   ```

6. View in the dashboard: http://localhost:8082

## Expected Output

The chaining orchestration greets Tokyo, Seattle, and London sequentially:
```json
["Hello Tokyo!", "Hello Seattle!", "Hello London!"]
```

The fan-out/fan-in orchestration greets all five cities in parallel and returns combined results:
```json
["Hello Tokyo!", "Hello Seattle!", "Hello London!", "Hello Paris!", "Hello Berlin!"]
```

## Using a Deployed Scheduler (Azure)

To use a Durable Task Scheduler in Azure instead of the emulator:

1. Set the connection string in `local.settings.json`:
   ```json
   {
     "Values": {
       "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=<your-scheduler-endpoint>;TaskHub=<your-taskhub>;Authentication=ManagedIdentity"
     }
   }
   ```

2. Run the sample using the same commands as above.

See the [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler) for setup instructions.

## Code Walkthrough

- **SayHello/** — Activity function that returns a greeting for a given city name.
- **ChainingOrchestration/** — Orchestrator that calls `SayHello` three times in sequence.
- **FanOutFanInOrchestration/** — Orchestrator that calls `SayHello` for five cities in parallel.
- **StartChaining/** — HTTP trigger that starts the chaining orchestration.
- **StartFanOutFanIn/** — HTTP trigger that starts the fan-out/fan-in orchestration.

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

## Learn More

- [Durable Functions PowerShell Developer Guide](https://learn.microsoft.com/azure/azure-functions/durable/quickstart-powershell-vscode)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
