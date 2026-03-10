# Bounded Coordinator — Durable Task SDK (.NET)

.NET | Durable Task SDK

## Description

Demonstrates a **bounded coordinator** pattern: a parent orchestration that fans out
child work in bounded batches and resets its history via `ContinueAsNew` after each
batch. This prevents unbounded history growth that can occur with long-lived
coordinator/message-pump orchestrations.

This pattern is important because:
- Long-lived coordinators accumulate history events on every replay
- Without periodic resets, history can grow to tens of thousands of events
- Large histories cause increasingly expensive replays and can lead to
  persistence failures in backing stores

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Start the worker (in one terminal):
   ```bash
   cd Worker
   dotnet run
   ```

3. Start the client (in another terminal):
   ```bash
   cd Client
   dotnet run
   ```

4. View the orchestration in the dashboard: http://localhost:8082

## How It Works

1. The coordinator reads a **bounded batch** of items via an activity
2. It fans out child sub-orchestrations for each item in the batch
3. It **waits for all children** to complete
4. If more work remains, it calls `ContinueAsNew` with compact carry-forward state
5. The orchestration restarts with a clean history and processes the next batch

## Anti-Pattern: Unbounded Coordinator

The following pattern looks similar but has a critical difference — it does **not**
call `ContinueAsNew`, so its history grows without bound:

```csharp
// BAD: This coordinator never resets its history
public override async Task RunAsync(TaskOrchestrationContext context, object? input)
{
    while (true)
    {
        var event = await context.WaitForExternalEvent<WorkItem>("new-item");
        await context.CallSubOrchestrationAsync("ProcessItem", event);
        // History grows by several events each iteration and is never reset
    }
}
```

## Learn More

- [Eternal Orchestrations](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
