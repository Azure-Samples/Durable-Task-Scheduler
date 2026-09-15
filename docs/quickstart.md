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
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
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
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
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
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler

# Run the sample
cd samples/durable-task-sdks/java/function-chaining
./gradlew runChainingPattern
```

### Go

**Requires:** [Go 1.25.0 or later](https://go.dev/dl/). The samples share one module using `github.com/microsoft/durabletask-go` **v1.0.0-beta.1**.

```bash
# Clone the repo (skip if already cloned)
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler

# Download dependencies once for all Go samples
cd samples/durable-task-sdks/go
go mod download

# Start the worker and client, verify the result, and exit
go run ./function-chaining
```

No second terminal or Azure account is needed. The default connection is `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`. If `DTS_CONNECTION_STRING` is already set, unset it or set it to that emulator connection string before running.

Run any of the [20 Go samples](../samples/durable-task-sdks/go) from the same module with `go run ./<sample-name>`, or from its directory with `go run .`. Check each README for additional feature-specific prerequisites. Go support is for the standalone SDK, not Durable Functions or Microsoft Agent Framework.

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

## Connect the Go Samples to Azure

Use an existing Azure Durable Task Scheduler and task hub, or provision them by following the [scheduler setup guide](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler). Grant the identity used by your application the **Durable Task Data Contributor** role at the appropriate scheduler or task-hub scope; see [identity-based access](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity).

Use a dedicated Go test hub when validating the full suite. The scheduled-tasks sample uses Go-owned system state; do not run Python/.NET schedule workers against the same schedule entities or assume cross-SDK schedule interoperability.

From `samples/durable-task-sdks/go`, authenticate and set the connection string:

```bash
az login
export DTS_CONNECTION_STRING='Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure'
go run ./function-chaining
```

Replace the placeholders with your scheduler endpoint host and task hub. `DefaultAzure` uses the Azure Identity default credential chain; use `Authentication=AzureCLI` to explicitly select the signed-in CLI identity. Production hosts can use managed or workload identity as described in the [Go SDK connection guide](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/durabletaskscheduler/README.md#configuration).

Scheduler and blob-storage configuration are independent. The Go large-payload and history-export samples default to local **Azurite**, even when `DTS_CONNECTION_STRING` points to Azure. A live scheduler with Azurite validates worker-side storage integration, not Azure Blob Storage. Follow the sample READMEs for storage configuration.

**History export requires an isolated task hub**, in the emulator or Azure, with no other export workers and no unrelated workloads completing during the sample's export window. The export query supports completion-time/status filters, not instance-ID prefix, name, or tag filters. For both emulator and Azure runs, set `HISTORY_EXPORT_ISOLATED_TASKHUB=1` only after confirming that the configured hub is isolated. This flag acknowledges isolation; it does not create a hub or isolate an existing one. Do not run history-export alongside other samples on a shared hub. Sample cleanup retains source histories and exported blobs.

The sample's [implemented ownership guards](../samples/durable-task-sdks/go/history-export/ownership.go) reject pages containing unowned instance IDs, direct unowned metadata/history reads, and writes outside the owned instance/container/prefix. The entire completion window is preflighted through the source guard before job creation. [Offline unit tests](../samples/durable-task-sdks/go/history-export/main_test.go), including `TestOwnershipGuardNeverReadsUnrelatedHistory` and `TestStorageOwnershipGuard`, verify these protections. Unit-test success does not establish emulator or live Azure backend validation.

These steps connect a locally running Go process to Azure. They do **not** deploy the worker. The Go samples do not include Azure Developer CLI (`azd`) templates or automated Container Apps/AKS deployment; choose and configure your hosting environment separately.

Connecting to Azure DTS also does not enable real AI providers. The Go agent demonstrations use explicit echo/synthetic fixtures by default; follow their READMEs for optional real-provider configuration and verification boundaries.

## Next Steps

| What to do | Link |
|-----------|------|
| Explore all patterns | [Orchestration Patterns Guide](./patterns.md) |
| Browse all samples | [Sample Catalog](../samples/README.md) |
| Explore the Go SDK | [Go Samples](../samples/durable-task-sdks/go) |
| Learn about the Durable Task Scheduler | [Official Documentation](https://aka.ms/dts-documentation) |
| Add OpenTelemetry tracing | [Observability Guide](./observability.md) |
| Deploy to Azure | [Azure deployment guide](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler) |
