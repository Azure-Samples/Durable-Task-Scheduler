# Sub-Orchestrations — Durable Task SDK (.NET)

.NET | Durable Task SDK

## Description

Demonstrates **sub-orchestrations** using the Durable Task SDK. A parent orchestration processes an order by delegating each line item to a child orchestration, which handles validation, pricing, and inventory reservation independently.

This pattern is useful for:
- Breaking complex workflows into smaller, reusable pieces
- Processing collections where each item has multi-step logic
- Isolating failure domains within a larger workflow

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

4. View the orchestration hierarchy in the dashboard: http://localhost:8082

## How It Works

1. The **parent orchestration** receives an order with multiple line items
2. For each line item, it calls a **child orchestration** as a sub-orchestration
3. Each child orchestration runs independently: validates the item, calculates the price, and reserves inventory
4. The parent collects all results and returns the complete order summary
5. If any child fails, the parent can handle the error gracefully

## Learn More

- [Sub-orchestrations](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-sub-orchestrations)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
