# Hello Cities — Durable Functions Python Quickstart

Python | Durable Functions

## Description

This quickstart demonstrates Durable Functions with Python (v2 programming model) using the Durable Task Scheduler backend. It includes two patterns:

1. **Function Chaining** — An orchestration that calls three "say hello" activities sequentially
2. **Fan-out/Fan-in** — An orchestration that greets multiple cities in parallel and aggregates results

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

## Quick Run

1. Start the emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Create a virtual environment and install dependencies:
   ```bash
   cd samples/durable-functions/python/hello-cities
   python -m venv .venv
   source .venv/bin/activate  # On Windows: .venv\Scripts\activate
   pip install -r requirements.txt
   ```

3. Run the function app:
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

## Learn More

- [Durable Functions Python API Reference](https://learn.microsoft.com/python/api/azure-functions-durable/azure.durable_functions)
- [Durable Functions Python Quickstart](https://learn.microsoft.com/azure/azure-functions/durable/quickstart-python-vscode)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
