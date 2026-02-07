# OpenTelemetry Distributed Tracing

Python | Durable Task SDK

## Description

This sample demonstrates how to add OpenTelemetry distributed tracing to a Durable Task SDK Python application. Traces are exported to Jaeger for visualization, allowing you to see the full flow of orchestrations and activities.

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/)

## Quick Run

1. Start the infrastructure (emulator + Jaeger):
   ```bash
   docker compose up -d
   ```

2. Install dependencies and start the worker:
   ```bash
   python -m venv venv
   source venv/bin/activate  # Windows: venv\Scripts\activate
   pip install -r requirements.txt
   python worker.py
   ```

3. In a new terminal, run the client:
   ```bash
   source venv/bin/activate
   python client.py
   ```

4. View traces:
   - **Jaeger UI:** http://localhost:16686 (search for service `durable-worker`)
   - **DTS Dashboard:** http://localhost:8082

## What You'll See

The Jaeger UI shows the complete trace for each orchestration — the parent orchestration span with child spans for each activity, including timing and any errors. This helps you:

- Identify slow activities
- See the sequential flow of function chaining
- Correlate traces across distributed services
- Debug failures with full context

## Learn More

- [Observability Guide](../../../../docs/observability.md)
- [OpenTelemetry Python docs](https://opentelemetry.io/docs/languages/python/)
- [Durable Task Scheduler Dashboard](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dashboard)
