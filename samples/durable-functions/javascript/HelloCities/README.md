# Hello Cities — Durable Functions JavaScript Quickstart

JavaScript | Durable Functions

## Description

This quickstart demonstrates Durable Functions with JavaScript (Node.js v4 programming model) using the Durable Task Scheduler backend. It includes two patterns:

1. **Function Chaining** — An orchestration that calls three "say hello" activities sequentially
2. **Fan-out/Fan-in** — An orchestration that greets multiple cities in parallel and aggregates results

## Prerequisites

1. [Node.js 18+](https://nodejs.org/)
2. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

## Quick Run

1. Start the emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Install dependencies and run:
   ```bash
   cd samples/durable-functions/javascript/HelloCities
   npm install
   func start
   ```

3. Trigger the function chaining orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartChaining
   ```

4. Trigger the fan-out/fan-in orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartFanOutFanIn
   ```

5. View in the dashboard: http://localhost:8082

## Expected Output

The chaining orchestration greets Tokyo, Seattle, and London sequentially:
```json
["Hello Tokyo!", "Hello Seattle!", "Hello London!"]
```

The fan-out/fan-in orchestration greets all five cities in parallel and returns combined results.

## Learn More

- [Durable Functions JavaScript API Reference](https://learn.microsoft.com/javascript/api/durable-functions/)
- [Durable Functions JavaScript Quickstart](https://learn.microsoft.com/azure/azure-functions/durable/quickstart-js-vscode)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
