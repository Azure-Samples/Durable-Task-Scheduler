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

Clone the repository, then open each terminal at the repository root:

```bash
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler
```

### .NET

**Requires:** [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0)

Start the worker in Terminal 1:

```bash
dotnet run --project samples/durable-task-sdks/dotnet/FunctionChaining/Worker
```

Run the client in Terminal 2:

```bash
dotnet run --project samples/durable-task-sdks/dotnet/FunctionChaining/Client
```

### Python

**Requires:** [Python 3.9+](https://www.python.org/downloads/)

Set up a virtual environment and start the worker in Terminal 1:

```bash
cd samples/durable-task-sdks/python/function-chaining
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
python worker.py
```

Activate the same virtual environment and run the client in Terminal 2:

```bash
cd samples/durable-task-sdks/python/function-chaining
source venv/bin/activate  # Windows: venv\Scripts\activate
python client.py
```

### Java

**Requires:** [Java 21](https://adoptium.net/)

This sample starts the worker and client in the same process:

```bash
cd samples/durable-task-sdks/java/function-chaining
./gradlew runChainingPattern
```

If the Gradle wrapper is not executable, run `chmod +x gradlew` first.

### JavaScript / TypeScript SDK (preview)

**Requires:** [Node.js 22+](https://nodejs.org/). The [SDK](https://github.com/microsoft/durabletask-js) supports JavaScript and TypeScript; this sample uses JavaScript.

Install dependencies and start the worker in Terminal 1:

```bash
cd samples/durable-task-sdks/javascript/function-chaining
npm install
npm run worker
```

Run the client in Terminal 2:

```bash
cd samples/durable-task-sdks/javascript/function-chaining
npm run client
```

Go samples in this repository are coming soon. See the [Go SDK repository](https://github.com/microsoft/durabletask-go) for the beta SDK and its own samples.

## Step 3: View in the Dashboard

1. Open [http://localhost:8082](http://localhost:8082) in your browser
2. Click on the **default** task hub
3. You'll see your orchestration instance(s) in the list
4. Click on an instance to view execution details: each activity, its input/output, and timing

## What Just Happened?

You ran a **function chaining** orchestration, a sequential workflow where:

1. Activity A processed some input and passed its output to...
2. Activity B, which transformed it and passed it to...
3. Activity C, which produced the final result

The orchestration is **durable**: if the process had crashed at any point, it would have automatically resumed from where it left off.

## Next Steps

| What to do | Link |
|-----------|------|
| Explore all patterns | [Orchestration Patterns Guide](./patterns.md) |
| Browse all samples | [Sample Catalog](../samples/README.md) |
| Learn about the Durable Task Scheduler | [Official Documentation](https://aka.ms/dts-documentation) |
| Add OpenTelemetry tracing | [Observability Guide](./observability.md) |
| Deploy to Azure | [Azure deployment guide](https://learn.microsoft.com/azure/durable-task/scheduler/develop-with-durable-task-scheduler) |
