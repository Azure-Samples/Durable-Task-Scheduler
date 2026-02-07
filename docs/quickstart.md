# ⚡ Quickstart: Your First Durable Orchestration

Get a durable orchestration running locally in under 5 minutes. No Azure subscription needed.

## Prerequisites

- [Docker](https://www.docker.com/products/docker-desktop/) installed and running

## Step 1: Start the Emulator

The Durable Task Scheduler emulator runs the full scheduler experience locally in Docker, including a monitoring dashboard.

```bash
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
```

Verify it's running by opening the dashboard: [http://localhost:8082](http://localhost:8082)

## Step 2: Run a Sample

Choose your language and follow the instructions:

### .NET

**Requires:** [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)

```bash
# Clone the repo
git clone https://github.com/Azure/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler

# Start the worker (Terminal 1)
cd samples/durable-task-sdks/dotnet/FunctionChaining/Worker
dotnet run

# Run the client (Terminal 2)
cd samples/durable-task-sdks/dotnet/FunctionChaining/Client
dotnet run
```

### Python

**Requires:** [Python 3.9+](https://www.python.org/downloads/)

```bash
# Clone the repo
git clone https://github.com/Azure/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler

# Set up environment
cd samples/durable-task-sdks/python/function-chaining
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt

# Start the worker (Terminal 1)
python worker.py

# Run the client (Terminal 2)
python client.py
```

### Java

**Requires:** [Java 8+](https://adoptium.net/)

```bash
# Clone the repo
git clone https://github.com/Azure/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler

# Run the sample
cd samples/durable-task-sdks/java/function-chaining
./gradlew runChainingPattern
```

## Step 3: View in the Dashboard

1. Open [http://localhost:8082](http://localhost:8082) in your browser
2. Click on the **default** task hub
3. You'll see your orchestration instance(s) in the list
4. Click on an instance to view execution details — each activity, its input/output, and timing

## What Just Happened?

You ran a **function chaining** orchestration — a sequential workflow where:

1. Activity A processed some input and passed its output to...
2. Activity B, which transformed it and passed it to...
3. Activity C, which produced the final result

The orchestration is **durable** — if the process had crashed at any point, it would have automatically resumed from where it left off.

## Next Steps

| What to do | Link |
|-----------|------|
| Explore all patterns | [Orchestration Patterns Guide](./patterns.md) |
| Browse all samples | [Sample Catalog](../samples/README.md) |
| Learn about the Durable Task Scheduler | [Official Documentation](https://aka.ms/dts-documentation) |
| Add OpenTelemetry tracing | [Observability Guide](./observability.md) |
| Deploy to Azure | [Azure deployment guide](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler) |
