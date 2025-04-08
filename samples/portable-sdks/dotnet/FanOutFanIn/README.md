# Durable Task Scheduler Fan-Out Fan-In Sample

This sample demonstrates how to implement the Fan-Out Fan-In pattern using the .NET Durable Task SDK with the Azure Durable Task Scheduler backend. It consists of two containerized microservices running on .NET 8.

## Architecture

Below is an architecture diagram illustrating the Fan-Out Fan-In pattern as implemented in this sample:

```
                                  +-------------------+
                                  |                   |
                                  |     HTTP Client   |
                                  |                   |
                                  +--------+----------+
                                           |
                                           | HTTP Request
                                           v
+------------------+              +--------+----------+
|                  |              |                   |
|  Azure Durable   |<------------>|   ClientService   |
|  Task Scheduler  |              |                   |
|                  |              +-------------------+
+--------+---------+                       ^
         |                                 |
         |                                 | Orchestration
         |                                 | Results
         |                                 |
         |                                 |
         |                                 |
         |                                 |
         |                                 |
         |                                 |
         |                                 |
         v                                 |
+--------+-----------------------------+   |
|                                      |   |
|          WorkerService               |   |
|   +----------------------------+     |   |
|   |                            |     |   |
|   |       HelloWorld           |     |   |
|   |      Orchestration         +-----+---+
|   |                            |     |
|   +-----+--------+--------+----+     |
|         |        |        |          |
|    Fan-Out       |        |          |
|         |        |        |          |
|         v        v        v          |
|    +----+--+ +---+--+ +---+--+       |
|    |       | |      | |      |       |
|    | Say   | | Say  | | Say  |       |
|    | Hello | | Hello| | Hello|       |
|    |       | |      | |      |       |
|    +-------+ +------+ +------+       |
|        ^                             |
|        |         Fan-In              |
|        +-----------------------------+
|                                      |
+--------------------------------------+

+------------------------------------------|
|                                          |
|  Legend:                                 |
|  -------                                 |
|  ➜ Request/Response Flow                 |
|  ↔ Service Communication                 |
|                                          |
+------------------------------------------+
```

The sample is structured as follows:

- **ClientService**: ASP.NET Core Web API that exposes endpoints to start and manage orchestrations
- **WorkerService**: ASP.NET Core service that processes the activities and implements the orchestration logic
- **Durable Task Scheduler**: The Azure backend service that manages the orchestration state and messaging

The Fan-Out Fan-In pattern flow:
1. Client makes HTTP request to the ClientService
2. ClientService creates one or more HelloWorld orchestrations
3. Each orchestration fans out to multiple parallel SayHello activities
4. The WorkerService processes each activity independently
5. Results from all activities are fanned back in to the orchestration
6. The aggregated results are stored in the Durable Task Scheduler
7. Client can query the orchestration status and results

Both services use the Durable Task SDK with the Azure Managed backend for orchestration management:

```csharp
// In ClientService/Program.cs
builder.Services.AddDurableTaskClient(clientBuilder =>
{
    clientBuilder.UseDurableTaskScheduler(connectionString);
});

// In WorkerService/Program.cs
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    workerBuilder.UseDurableTaskScheduler(connectionString);
    // Register orchestrations and activities...
});
```

## Prerequisites

- .NET 8 SDK
- Docker (for containerized deployment)
- Azure Durable Task Scheduler instance (or local development storage)

## Configuration

Both services use a connection string to connect to the Durable Task Scheduler backend. You can set the connection string using an environment variable:

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

### Using Docker

The simplest way to run the sample is using Docker:

```bash
# Build and run the Docker containers
./build-docker.sh
```

Alternatively, you can use Azure Developer CLI to deploy to Azure:

```bash
azd up
```

### Running Locally

#### Start the Worker Service

```bash
cd WorkerService
dotnet run
```

The Worker Service will start and wait for orchestration tasks.

#### Start the Client Service

```bash
cd ClientService
dotnet run
```

The Client Service will start and expose HTTP endpoints to create and manage orchestrations.

## Using the Sample

### Create a Fan-Out Fan-In Orchestration

Send a POST request to the Client Service to start a fan-out fan-in pattern test:

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"iterations":10,"parallelActivities":5,"parallelOrchestrations":1}' \
  http://localhost:8080/api/orchestrations
```

This will create a new orchestration that:
- Runs 10 iterations
- In each iteration, fans out to 5 parallel activities (SayHello)
- Fans in the results from all activities
- You can also create multiple parallel orchestrations

### Check Orchestration Status

You can check the status of your orchestration using the returned instance ID:

```bash
curl http://localhost:8080/api/orchestrations/{instanceId}
```

You can also check multiple orchestrations at once:

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '["instance1-id", "instance2-id", "instance3-id"]' \
  http://localhost:8080/api/orchestrations/status
```

### Check Service Status

You can verify both services are running with:

```bash
# Client Service status
curl http://localhost:8080/status

# Worker Service status
curl http://localhost:8080/status
```

## Fan-Out Fan-In Parameters

The fan-out fan-in test accepts the following parameters:

- **iterations**: The number of sequential iterations to run
- **parallelActivities**: The number of parallel activities to fan out to in each iteration
- **parallelOrchestrations**: The number of parallel orchestrations to create (defaults to 1)

## Key Components in Detail

### 1. Client Service (HTTP API Layer)

The ClientService (`ClientService/Program.cs`) is responsible for handling HTTP requests and scheduling orchestrations. Key components include:

#### API Endpoints

```csharp
// Start new orchestration(s)
app.MapPost("/api/orchestrations", async ([FromServices] DurableTaskClient client, [FromBody] FanOutFanInRequest request) =>
{
    // The code schedules orchestrations based on request parameters
    var tasks = new List<Task<string>>();
    
    for (int i = 0; i < request.ParallelOrchestrations; i++)
    {
        // Schedule orchestration with the HelloWorld function
        tasks.Add(client.ScheduleNewOrchestrationInstanceAsync(
            "HelloWorld", 
            new FanOutFanInOrchestrationInput
            {
                Iterations = request.Iterations,
                ParallelActivities = request.ParallelActivities
            }));
    }
    
    // Return accepted response with orchestration IDs
    instanceIds = (await Task.WhenAll(tasks)).ToList();
    return Results.Accepted($"/api/orchestrations/status", new { 
        count = instanceIds.Count,
        instanceIds = instanceIds 
    });
});

// Check status of an orchestration
app.MapGet("/api/orchestrations/{instanceId}", async ([FromServices] DurableTaskClient client, string instanceId) =>
{
    var instance = await client.GetInstanceAsync(instanceId);
    // Return orchestration status details
});

// Additional endpoints for checking multiple statuses and service health...
```

#### Input and Result Models

The ClientService defines models for API requests and responses in `ClientService/Models/`:

```csharp
// ClientInputModels.cs
public class FanOutFanInRequest
{
    public int Iterations { get; set; } = 10;
    public int ParallelActivities { get; set; } = 5;
    public int ParallelOrchestrations { get; set; } = 1;
}

// ClientResultModels.cs
public class OrchestrationStatusResponse
{
    public string InstanceId { get; set; }
    public string Status { get; set; }
    public DateTime CreatedAt { get; set; }
    public string Output { get; set; }
    // Additional properties...
}
```

### 2. Worker Service (Orchestration & Activity Logic)

The WorkerService (`WorkerService/Program.cs`) handles the execution of orchestrations and activities. Key components include:

#### Task Registration

The worker service uses a fluent API to register orchestrations and activities:

```csharp
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    // Configure the worker to use the Durable Task Scheduler backend
    workerBuilder.UseDurableTaskScheduler(connectionString);
    
    // Register all tasks (orchestrations and activities)
    workerBuilder.AddTasks(registry =>
    {
        // Register the HelloWorld orchestration
        registry.AddOrchestratorFunc<FanOutFanInOrchestrationInput, FanOutFanInTestResult>(
            "HelloWorld", 
            async (ctx, input) => {
                var orchestration = new HelloWorld(logger);
                return await orchestration.RunAsync(ctx, input);
            });

        // Register the SayHello activity
        registry.AddActivityFunc<ActivityInput, ActivityResult>(
            "SayHello",
            async (ctx, input) => {
                var activity = new SayHello(logger);
                return await activity.RunAsync(ctx, input);
            });
    });
});
```

#### HelloWorld Orchestration Implementation

The orchestration implementation (`WorkerService/Orchestrations/FanOutFanInOrchestration.cs`) is the heart of the Fan-Out Fan-In pattern:

```csharp
public async Task<FanOutFanInTestResult> RunAsync(TaskOrchestrationContext context, FanOutFanInOrchestrationInput input)
{
    var stopwatch = Stopwatch.StartNew();
    var results = new List<ActivityResult>();
    
    // Run multiple iterations of parallel activities
    for (int i = 0; i < input.Iterations; i++)
    {
        var tasks = new List<Task<ActivityResult>>();
        
        // Fan-Out: Create multiple parallel activities
        for (int j = 0; j < input.ParallelActivities; j++)
        {
            var task = context.CallActivityAsync<ActivityResult>(
                "SayHello",
                new ActivityInput { 
                    IterationNumber = i, 
                    ActivityNumber = j 
                });
            tasks.Add(task);
        }
        
        // Wait for all parallel activities to complete
        await Task.WhenAll(tasks);
        
        // Fan-In: Collect results
        results.AddRange(tasks.Select(t => t.Result));
    }
    
    stopwatch.Stop();
    
    // Return aggregated results
    return new FanOutFanInTestResult
    {
        TotalActivities = input.Iterations * input.ParallelActivities,
        ElapsedTimeMs = stopwatch.ElapsedMilliseconds,
        AverageActivityTimeMs = results.Average(r => r.ProcessingTimeMs),
        Results = results
    };
}
```

Key aspects of this implementation:
1. **Iterations Loop**: Runs multiple iterations sequentially
2. **Fan-Out**: Creates multiple parallel activities in each iteration
3. **Task.WhenAll**: Waits for all parallel activities to complete (synchronization point)
4. **Fan-In**: Collects results from all parallel activities
5. **Performance Tracking**: Records execution time and calculates statistics

#### SayHello Activity Implementation

The activity implementation (`WorkerService/Activities/FanOutFanInActivity.cs`) represents the work performed in parallel:

```csharp
public Task<ActivityResult> RunAsync(TaskActivityContext context, ActivityInput input)
{
    var stopwatch = Stopwatch.StartNew();
    
    try
    {
        // Simulate some work
        string output = "Hello World";
        
        stopwatch.Stop();
        
        // Return activity result with timing information
        return Task.FromResult(new ActivityResult
        {
            IterationNumber = input.IterationNumber,
            ActivityNumber = input.ActivityNumber,
            ProcessingTimeMs = stopwatch.ElapsedMilliseconds,
            Output = output
        });
    }
    catch (Exception ex)
    {
        stopwatch.Stop();
        _logger?.LogError(ex, "Activity failed");
        throw; // Re-throw to let the Durable Task Framework handle it
    }
}
```

#### Model Classes

The Worker Service defines models for orchestration inputs and outputs:

```csharp
// WorkerService/Models/OrchestrationModels.cs
public class FanOutFanInOrchestrationInput
{
    public int Iterations { get; set; }
    public int ParallelActivities { get; set; }
}

public class FanOutFanInTestResult
{
    public int TotalActivities { get; set; }
    public long ElapsedTimeMs { get; set; }
    public double AverageActivityTimeMs { get; set; }
    public List<ActivityResult> Results { get; set; } = new();
}

// WorkerService/Models/ActivityModels.cs
public class ActivityInput
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
}

public class ActivityResult
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
    public long ProcessingTimeMs { get; set; }
    public string Output { get; set; } = string.Empty;
}
```

### 3. Azure Durable Task Scheduler Integration

Both services connect to the Azure Durable Task Scheduler service, which handles:
- Persisting orchestration state
- Delivering messages between services
- Managing activity executions
- Handling retries and error handling
- Providing monitoring and diagnostics

## Containerization

Both services are containerized using Docker. The Dockerfiles demonstrate how to package .NET applications that use the Durable Task SDK:

- `ClientService/Dockerfile`
- `WorkerService/Dockerfile`

When deployed to Azure, the `azure.yaml` file configures how these containers are deployed to Azure Container Apps.

## Advanced Features and Best Practices

- **Robust Logging**: Both services use structured logging with correlation IDs to track orchestrations and activities
- **Error Handling**: Activities use try-catch blocks to properly handle and report errors
- **Performance Tracking**: Execution times are tracked and reported
- **Containerization**: Services are designed to run in containers for easy deployment
- **Configurability**: Connection strings and other settings are externalized
- **Health Endpoints**: Both services provide status endpoints for monitoring

## Conclusion

This sample demonstrates how to implement the Fan-Out Fan-In pattern using the Durable Task SDK with Azure Durable Task Scheduler. It showcases parallel processing, orchestration coordination, and proper error handling in a distributed system architecture.