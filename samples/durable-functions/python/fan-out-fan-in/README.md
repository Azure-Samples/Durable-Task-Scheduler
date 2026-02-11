# Fan-out/Fan-in with Durable Functions (Python)

Python | Durable Functions

## Description

This sample demonstrates the fan-out/fan-in pattern using Azure Durable Functions with Python. It processes a batch of work items in parallel and aggregates the results — a fundamental pattern for parallel data processing.

The orchestration:
1. Generates a list of work items
2. Fans out by scheduling parallel activity executions for each item
3. Waits for all activities to complete
4. Aggregates and returns the combined results

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/)
3. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)

## Quick Run

1. Start the emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Set up and run:
   ```bash
   python -m venv venv
   source venv/bin/activate
   pip install -r requirements.txt
   func start
   ```

3. Trigger the orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartFanOutFanIn
   ```

4. View in dashboard: http://localhost:8082

## Learn More

- [Fan-out/Fan-in Pattern](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-cloud-backup)
- [Durable Functions Python Guide](https://learn.microsoft.com/azure/azure-functions/durable/quickstart-python-vscode)
