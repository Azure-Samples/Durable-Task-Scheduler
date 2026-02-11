# Eternal Orchestrations — Durable Task SDK (.NET)

.NET | Durable Task SDK

## Description

Demonstrates **eternal orchestrations** using the Durable Task SDK. An orchestration that runs indefinitely by periodically performing work and restarting itself with `ContinueAsNew`, which clears its history to prevent unbounded growth.

This pattern is useful for:
- Periodic data cleanup or aggregation
- Heartbeat or health-check monitoring
- Background jobs that run on a schedule
- Any long-running loop that must survive restarts

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
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

4. View the eternal orchestration in the dashboard: http://localhost:8082

## How It Works

1. The orchestration performs a cleanup task (simulated)
2. It logs the iteration count and result
3. It waits using a durable timer (e.g., 10 seconds)
4. It calls `ContinueAsNew` with an incremented counter, which restarts the orchestration with a clean history
5. This loop continues indefinitely until explicitly terminated

## Learn More

- [Eternal Orchestrations](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
