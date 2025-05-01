using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using System.Collections.Generic;
using System.Text.Json;

// Configure logging
using ILoggerFactory loggerFactory = LoggerFactory.Create(builder =>
{
    builder.AddConsole();
    builder.SetMinimumLevel(LogLevel.Information);
});

ILogger<Program> logger = loggerFactory.CreateLogger<Program>();
logger.LogInformation("Starting Fan-Out Fan-In Pattern - Parallel Processing Client");

// Get environment variables for endpoint and taskhub with defaults
string endpoint = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHubName = Environment.GetEnvironmentVariable("TASKHUB") ?? "default";

// Split the endpoint if it contains authentication info
string hostAddress = endpoint;
if (endpoint.Contains(';'))
{
    hostAddress = endpoint.Split(';')[0];
}

// Determine if we're connecting to the local emulator
bool isLocalEmulator = endpoint == "http://localhost:8080";

// Construct a proper connection string with authentication
string connectionString;
if (isLocalEmulator)
{
    // For local emulator, no authentication needed
    connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=None";
    logger.LogInformation("Using local emulator with no authentication");
}
else
{
    // For Azure, use DefaultAzure authentication
    connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=DefaultAzure";
    logger.LogInformation("Using Azure endpoint with DefaultAzure authentication");
}

logger.LogInformation("Using endpoint: {Endpoint}", endpoint);
logger.LogInformation("Using task hub: {TaskHubName}", taskHubName);
logger.LogInformation("Host address: {HostAddress}", hostAddress);
logger.LogInformation("Connection string: {ConnectionString}", connectionString);
logger.LogInformation("This client submits a list of work items for parallel processing");

// Create the client using DI service provider
ServiceCollection services = new ServiceCollection();
services.AddLogging(builder => builder.AddConsole());

// Register the client
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

// Create a list of work items to process in parallel
List<string> workItems = new List<string>
{
    "Task1",
    "Task2",
    "Task3",
    "LongerTask4",
    "VeryLongTask5"
};

logger.LogInformation("Starting parallel processing orchestration with {Count} work items", workItems.Count);
logger.LogInformation("Work items: {WorkItems}", JsonSerializer.Serialize(workItems));

// Schedule the orchestration
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "ParallelProcessingOrchestration", 
    workItems);

logger.LogInformation("Started orchestration with ID: {InstanceId}", instanceId);

// Wait for orchestration to complete
logger.LogInformation("Waiting for orchestration to complete...");

// Create a cancellation token source with timeout
using CancellationTokenSource cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));

// Wait for the orchestration to complete using built-in method
OrchestrationMetadata instance = await client.WaitForInstanceCompletionAsync(
    instanceId,
    getInputsAndOutputs: true,
    cts.Token);

logger.LogInformation("Orchestration completed with status: {Status}", instance.RuntimeStatus);

if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
{
    Dictionary<string, int>? results = instance.ReadOutputAs<Dictionary<string, int>>();
    logger.LogInformation("Processing results:");
    if (results != null)
    {
        foreach (KeyValuePair<string, int> result in results)
        {
            logger.LogInformation("Work item: {Item}, Result: {Result}", result.Key, result.Value);
        }
        
        logger.LogInformation("Total items processed: {Count}", results.Count);
    }
    else
    {
        logger.LogWarning("No results were returned from the orchestration.");
    }
}
else if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Failed)
{
    logger.LogError("Orchestration failed: {ErrorMessage}", instance.FailureDetails?.ErrorMessage);
}
