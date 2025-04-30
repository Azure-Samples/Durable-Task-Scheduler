# Autoscaling in Azure Container Apps Pattern

## Description of the Sample

This sample demonstrates how to implement autoscaling with the Azure Durable Task Scheduler using the .NET SDK in Azure Container Apps. The pattern showcases an orchestration workflow that can benefit from dynamically scaling worker instances based on load.

In this sample:
1. The orchestrator calls the `SayHelloActivity` which greets the user with their name
2. The result is passed to the `ProcessGreetingActivity` which adds to the greeting
3. The result is then passed to the `FinalizeResponseActivity` which completes the greeting
4. The final greeting message is returned to the client

This pattern is useful for:
- Creating scalable workflows that can handle varying loads
- Efficiently utilizing resources by scaling container instances up or down
- Building resilient systems that can respond to increased demand
- Implementing cost-effective processing for variable workloads

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Install Docker if it's not already installed.

2. Pull the Docker Image for the Emulator:
```bash
docker pull mcr.microsoft.com/dts/dts-emulator:v0.0.6
```

3. Run the Emulator:
```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:v0.0.6
```
Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: http://localhost:8080, taskhub: default). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

For production scenarios or when you're ready to deploy to Azure:

1. Create a Scheduler using the Azure CLI:
```bash
az durabletask scheduler create --resource-group <testrg> --name <testscheduler> --location <eastus> --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
```

2. Create Your Taskhub:
```bash
az durabletask taskhub create --resource-group <testrg> --scheduler-name <testscheduler> --name <testtaskhub>
```

3. Retrieve the Endpoint for the Scheduler from the Azure portal.

4. Set the Environment Variables:

   Bash:
   ```bash
   export TASKHUB=<taskhubname>
   export ENDPOINT=<taskhubEndpoint>
   ```

   PowerShell:
   ```powershell
   $env:TASKHUB = "<taskhubname>"
   $env:ENDPOINT = "<taskhubEndpoint>"
   ```

## Authentication

The sample includes smart detection of the environment and configures authentication automatically:

- For local development with the emulator (when endpoint is http://localhost:8080), no authentication is required.
- For Azure deployments, DefaultAzureCredential is used, which tries multiple authentication methods:
  - Managed Identity
  - Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
  - Azure CLI login
  - Visual Studio login
  - and more

The connection string is constructed dynamically based on the environment:
```csharp
// For local emulator
connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=None";

// For Azure
connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=DefaultAzureCredential";
```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, build the solution:
```bash
cd AutoscalingInACA
dotnet build
```

2. Start the worker in a terminal:
```bash
cd Worker
dotnet run
```
You should see output indicating the worker has started and registered the orchestration and activities.

3. In a new terminal, run the client:
```bash
cd Client
dotnet run
```

## Understanding the Code Structure

### Worker Project

The Worker project contains:

- **GreetingOrchestration.cs**: Defines the orchestrator and activity functions in a single file
- **Program.cs**: Sets up the worker host with proper connection string handling

#### Orchestration Implementation

The orchestration directly calls each activity in sequence using the standard `CallActivityAsync` method:

```csharp
public override async Task<string> RunAsync(TaskOrchestrationContext context, string name)
{
    // Step 1: Say hello to the person
    string greeting = await context.CallActivityAsync<string>(nameof(SayHelloActivity), name);
    
    // Step 2: Process the greeting
    string processedGreeting = await context.CallActivityAsync<string>(nameof(ProcessGreetingActivity), greeting);
    
    // Step 3: Finalize the response
    string finalResponse = await context.CallActivityAsync<string>(nameof(FinalizeResponseActivity), processedGreeting);
    
    return finalResponse;
}
```

Each activity is implemented as a separate class decorated with the `[DurableTask]` attribute:

```csharp
[DurableTask]
public class SayHelloActivity : TaskActivity<string, string>
{
    // Implementation details
}
```

The worker uses Microsoft.Extensions.Hosting for proper lifecycle management:
```csharp
var builder = Host.CreateApplicationBuilder();
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry => {
        registry.AddAllGeneratedTasks();
    })
    .UseDurableTaskScheduler(connectionString);
var host = builder.Build();
await host.StartAsync();
```

### Client Project

The Client project:

- Uses the same connection string logic as the worker
- Schedules an orchestration instance with a name input
- Waits for the orchestration to complete and displays the result
- Uses WaitForInstanceCompletionAsync for efficient polling

```csharp
var instance = await client.WaitForInstanceCompletionAsync(
    instanceId,
    getInputsAndOutputs: true,
    cts.Token);
```

## Understanding the Output

When you run the client, you should see:
1. The client starting an orchestration with a name input
2. The worker processing the chained activities
3. The client displaying the final result from the completed orchestration

Example output:
```
Starting greeting orchestration for name: GitHub Copilot
Started orchestration with ID: 7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b
Waiting for orchestration to complete...
Orchestration completed with status: Completed
Greeting result: Hello GitHub Copilot! It's nice to meet you. Welcome to the Durable Task Framework!
```

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The sequential execution of the activities
   - The input and output at each step
   - The time taken for each step

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details
