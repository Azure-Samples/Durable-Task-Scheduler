# Durable Task Scheduler Fan-Out Fan-In Sample

This sample demonstrates how to implement the Fan-Out Fan-In pattern using the Durable Task SDK with the Azure Durable Task Scheduler backend. It consists of two microservices:

1. **OrchestrationService**: Accepts HTTP requests and converts them into orchestrations
2. **WorkerService**: Implements the activity logic for the orchestrations

## Prerequisites

- .NET 8 SDK
- Azure Durable Task Scheduler instance (or local development storage)

## Configuration

Both services use a connection string to connect to the Durable Task Scheduler backend. By default, they use the local development storage setting:

```
UseDevelopmentStorage=true
```

You can set the connection string using an environment variable:

```bash
export DURABLE_TASK_CONNECTION_STRING="your-connection-string-here"
```

Or by updating the `appsettings.json` files in both services:

```json
{
  "DurableTaskScheduler": {
    "ConnectionString": "your-connection-string-here"
  }
}
```

## Running the Sample

### Start the Worker Service

```bash
cd WorkerService/WorkerService
dotnet run
```

The Worker Service will start and wait for orchestration tasks.

### Start the Orchestration Service

```bash
cd OrchestrationService/OrchestrationService
dotnet run
```

The Orchestration Service will start and expose HTTP endpoints to create and manage orchestrations.

## Using the Sample

### Create a Fan-Out Fan-In Orchestration

Send a POST request to the Orchestration Service to start a fan-out fan-in pattern test:

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"iterations":10,"parallelActivities":5,"parallelOrchestrations":1}' \
  http://localhost:5000/api/orchestrations
```

This will create a new orchestration that:
- Runs 10 iterations
- In each iteration, fans out to 5 parallel activities
- Fans in the results from all activities
- You can also create multiple parallel orchestrations

### Check Orchestration Status

You can check the status of your orchestration using the returned instance ID:

```bash
curl http://localhost:5000/api/orchestrations/{instanceId}
```

You can also check multiple orchestrations at once:

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '["instance1-id", "instance2-id", "instance3-id"]' \
  http://localhost:5000/api/orchestrations/status
```

### Check Service Status

You can verify both services are running with:

```bash
# Orchestration Service status
curl http://localhost:5000/status

# Worker Service status
curl http://localhost:5001/status
```

## Fan-Out Fan-In Parameters

The fan-out fan-in test accepts the following parameters:

- **iterations**: The number of sequential iterations to run
- **parallelActivities**: The number of parallel activities to fan out to in each iteration
- **parallelOrchestrations**: The number of parallel orchestrations to create (defaults to 1)

## Architecture

The sample is structured as follows:

- **OrchestrationService**: ASP.NET Core Web API that exposes endpoints to start and manage orchestrations
- **WorkerService**: ASP.NET Core service that processes the activities and implements the orchestration logic

Both services use the Durable Task SDK with the Azure Managed backend for orchestration management:

```csharp
// In OrchestrationService/Program.cs
builder.Services.AddDurableTaskClient("FanOutFanInClient", options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

// In WorkerService/Program.cs
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    workerBuilder.UseDurableTaskScheduler(connectionString);
    // Register orchestrations and activities...
});
```

## Performance Insights

This sample demonstrates the Fan-Out Fan-In pattern, which is useful for:

1. Processing data in parallel and then aggregating results
2. Executing multiple independent tasks concurrently
3. Scaling out workloads across multiple compute resources
4. Reducing overall execution time for parallelizable workloads