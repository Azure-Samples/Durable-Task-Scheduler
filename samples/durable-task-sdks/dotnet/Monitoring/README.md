# Monitor Pattern — Durable Task SDK (.NET)

.NET | Durable Task SDK

## Description

Demonstrates the **Monitor** pattern using the Durable Task SDK. The orchestration periodically checks a simulated job status, sleeping between polls, until the job completes or a timeout is reached.

This pattern is useful for:
- Polling external APIs or services
- Waiting for long-running jobs to complete
- Implementing flexible timeouts with periodic checks

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

4. View orchestration progress in the dashboard: http://localhost:8082

## How It Works

1. The orchestration starts monitoring a "job" with a unique ID
2. Each iteration calls the `CheckJobStatus` activity to poll the job state
3. If the job is complete, the orchestration returns the result
4. If not, it creates a durable timer to wait (e.g., 5 seconds) and then calls `ContinueAsNew` to restart with updated state
5. A timeout guard prevents infinite polling

## Learn More

- [Monitor Pattern](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview#monitoring)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
