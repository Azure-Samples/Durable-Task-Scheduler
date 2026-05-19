# Work Item Filtering with Durable Functions (.NET)

.NET | Durable Functions

## Description

Demonstrates the **work item filtering** feature for Durable Functions with the Durable Task Scheduler (DTS) backend. When multiple Function apps share the same DTS task hub, work item filtering ensures each app only receives work items for the functions it has registered — preventing dispatch failures.

This sample includes orchestrations, activities, entities, sub-orchestrations, and fan-out/fan-in patterns — all governed by work item filters.

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator and Azurite)
3. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Start Azurite (Azure Storage emulator):
   ```bash
   docker run --name azurite -d -p 10000:10000 -p 10001:10001 -p 10002:10002 mcr.microsoft.com/azure-storage/azurite
   ```

3. Start the Function app:
   ```bash
   func start
   ```

4. Trigger an orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/orchestrators/greeting
   ```

5. Test filter isolation — schedule an orchestration this app does NOT have:
   ```bash
   curl -X POST http://localhost:7071/api/start/SomeOtherOrchestration
   ```
   Check the status — it should stay `Pending` because no worker has `SomeOtherOrchestration` in its filter. Without filtering, this would fail with *"function doesn't exist"*.

## Expected Output

```
# Matching orchestration → Completed
{"name":"GreetingOrchestration","runtimeStatus":"Completed","output":"Hello, World!"}

# Unknown orchestration → stays Pending (filter isolation working)
{"name":"SomeOtherOrchestration","runtimeStatus":"Pending"}
```

## How It Works

The key configuration in [`host.json`](host.json):

```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING",
        "workItemFilteringEnabled": true
      }
    }
  }
}
```

When `workItemFilteringEnabled` is `true`:
1. The Durable Functions extension discovers registered orchestrators, activities, and entities during function indexing
2. These names are sent to DTS as `WorkItemFilters` on the `GetWorkItems` gRPC stream
3. DTS only dispatches work items that match the worker's registered functions
4. Unmatched work items stay in the DTS queue until a matching worker connects

No code changes are needed — filtering is automatic based on the functions registered in the app.

## Registered Functions

| Type          | Function                  | Description                         |
|---------------|---------------------------|-------------------------------------|
| Orchestration | `GreetingOrchestration`   | Simple activity call                |
| Orchestration | `FanOutOrchestration`     | Parallel fan-out to 3 activities    |
| Orchestration | `ParentOrchestration`     | Calls GreetingOrchestration as sub  |
| Orchestration | `CounterOrchestration`    | Interacts with CounterEntity        |
| Activity      | `SayHello`                | Returns a greeting string           |
| Entity        | `CounterEntity`           | Counter with Add/Reset/Get          |

## Multi-App Scenario

A sibling project [`WorkItemFiltering.AppB`](../WorkItemFiltering.AppB/) registers an entirely **different** set of functions (`OrdersOrchestration`, `ShipOrder` activity) against the **same** DTS task hub (`default`). The scheduler routes each work item to whichever app's filter matches.

Run both apps together with the included script:

```powershell
# From samples/durable-functions/dotnet/
.\run-both.ps1
```

The script builds both projects, starts App A on `:7071` and App B on `:7072`, and exercises five scenarios:

| # | Scenario                                                       | Expected            |
|---|----------------------------------------------------------------|---------------------|
| 1 | App A orchestration, scheduled from App A client               | `Completed`         |
| 2 | App B orchestration, scheduled from App B client               | `Completed`         |
| 3 | **Cross-app:** App B's `OrdersOrchestration` from App A client | `Completed` on B    |
| 4 | **Cross-app:** App A's `GreetingOrchestration` from App B client | `Completed` on A  |
| 5 | Orchestration neither app has registered                       | `Pending` (forever) |

Scenarios 3 and 4 are the point: the client app does not need to host the orchestrator. Without `workItemFilteringEnabled`, both apps would race to dispatch every work item and one would fail with *"function does not exist"*. With filtering on, the scheduler delivers only matching work to each app.

Helpful flags:

```powershell
.\run-both.ps1 -StartOnly   # leave both running for manual testing
.\run-both.ps1 -StopOnly    # kill any running func hosts
```

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

## Using a Deployed Scheduler (Azure)

To use a Durable Task Scheduler in Azure instead of the emulator:

1. Update `local.settings.json`:
   ```json
   {
     "Values": {
       "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=https://<your-scheduler>.durabletask.io;Authentication=ManagedIdentity"
     }
   }
   ```

2. Run the sample using the same commands above.

## Related Samples

- [WorkItemFilteringSplitActivities](../../../scenarios/WorkItemFilteringSplitActivities/) — Multi-worker scenario using Durable Task SDK
- [Fan-out/Fan-in (Python)](../../python/fan-out-fan-in/) — Fan-out pattern in Python
- [HelloCities (.NET)](../HelloCities/) — Basic Durable Functions quickstart

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-task-hubs)
- [Durable Functions patterns](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
