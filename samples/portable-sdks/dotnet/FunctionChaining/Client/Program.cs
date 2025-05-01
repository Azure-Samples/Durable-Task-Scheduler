using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using System.Text.Json;
using System.Collections.Generic;
using System.Threading.Tasks;
using System.Diagnostics;

// Configure logging
using ILoggerFactory loggerFactory = LoggerFactory.Create(builder =>
{
    builder.AddConsole();
    builder.SetMinimumLevel(LogLevel.Information);
});

ILogger<Program> logger = loggerFactory.CreateLogger<Program>();
logger.LogInformation("Starting Function Chaining Pattern - Greeting Client");

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
    // For Azure, use DefaultAzure - make sure TaskHub is included
    if (!endpoint.Contains("TaskHub="))
    {
        // Append the TaskHub parameter if it's not already in the connection string
        connectionString = $"{endpoint};TaskHub={taskHubName}";
    }
    else
    {
        connectionString = endpoint;
    }
    logger.LogInformation("Using Azure endpoint with DefaultAzure");
}

logger.LogInformation("Using endpoint: {Endpoint}", endpoint);
logger.LogInformation("Using task hub: {TaskHubName}", taskHubName);
logger.LogInformation("Host address: {HostAddress}", hostAddress);
logger.LogInformation("Connection string: {ConnectionString}", connectionString);
logger.LogInformation("This sample implements a simple greeting workflow with 3 chained activities");

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

// Create a name input for the greeting orchestration
string name = "User";
logger.LogInformation("Starting sequential orchestration scheduler - 20 orchestrations, 1 every 5 seconds");

// Set up orchestration parameters
const int TotalOrchestrations = 20;  // Total number of orchestrations to run
const int IntervalSeconds = 5;       // Time between orchestrations in seconds
var completedOrchestrations = 0;     // Track total completed orchestrations
var failedOrchestrations = 0;        // Track total failed orchestrations

// Run the main workflow to schedule and wait for all orchestrations
await RunSequentialOrchestrationsAsync();

logger.LogInformation("All orchestrations completed. Application shutting down.");

// Method to run orchestrations sequentially
async Task RunSequentialOrchestrationsAsync()
{
    // List to track all instance ids for monitoring
    var allInstanceIds = new List<string>(TotalOrchestrations);
    
    // Schedule each orchestration with delay between them
    for (int i = 0; i < TotalOrchestrations; i++)
    {
        // Create a unique instance ID
        string instanceName = $"{name}_{i+1}";
        logger.LogInformation("Scheduling orchestration #{Number} ({InstanceName})", i+1, instanceName);
        
        var stopwatch = Stopwatch.StartNew();
        try
        {
            // Schedule the orchestration
            string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
                "GreetingOrchestration", 
                instanceName);
                
            allInstanceIds.Add(instanceId);
            stopwatch.Stop();
            
            logger.LogInformation("Orchestration #{Number} scheduled in {ElapsedMs}ms with ID: {InstanceId}", 
                i+1, stopwatch.ElapsedMilliseconds, instanceId);
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error scheduling orchestration #{Number}", i+1);
        }
        
        // Wait before scheduling next orchestration (except for the last one)
        if (i < TotalOrchestrations - 1)
        {
            logger.LogInformation("Waiting {Seconds} seconds before scheduling next orchestration...", IntervalSeconds);
            await Task.Delay(TimeSpan.FromSeconds(IntervalSeconds));
        }
    }
    
    logger.LogInformation("All {Count} orchestrations scheduled. Waiting for completion...", allInstanceIds.Count);

    // Now wait for all orchestrations to complete
    foreach (string id in allInstanceIds)
    {
        try
        {
            OrchestrationMetadata instance = await client.WaitForInstanceCompletionAsync(
                id, getInputsAndOutputs: false, CancellationToken.None);
            
            if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
            {
                completedOrchestrations++;
                logger.LogInformation("Orchestration {Id} completed successfully", instance.InstanceId);
            }
            else if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Failed)
            {
                failedOrchestrations++;
                logger.LogError("Orchestration {Id} failed: {ErrorMessage}", 
                    instance.InstanceId, instance.FailureDetails?.ErrorMessage);
            }
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error waiting for orchestration {Id} completion", id);
        }
    }
    
    // Log final stats
    logger.LogInformation("FINAL RESULTS: {Completed} completed, {Failed} failed, {Total} total orchestrations", 
        completedOrchestrations, failedOrchestrations, allInstanceIds.Count);
}