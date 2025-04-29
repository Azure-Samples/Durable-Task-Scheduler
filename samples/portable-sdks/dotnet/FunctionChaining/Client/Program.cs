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
    // For Azure, use DefaultAzureCredential - make sure TaskHub is included
    if (!endpoint.Contains("TaskHub="))
    {
        // Append the TaskHub parameter if it's not already in the connection string
        connectionString = $"{endpoint};TaskHub={taskHubName}";
    }
    else
    {
        connectionString = endpoint;
    }
    logger.LogInformation("Using Azure endpoint with DefaultAzureCredential");
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
logger.LogInformation("Scheduling 2,000 orchestrations concurrently for name: {Name}", name);

// Create a stopwatch to measure performance
Stopwatch stopwatch = Stopwatch.StartNew();

// Create a list to store all instance IDs
List<string> instanceIds = new List<string>(2000);

// Schedule 2,000 orchestrations all at once
const int OrchestrationCount = 2000;
var scheduleTasks = new List<Task<string>>(OrchestrationCount);

for (int i = 0; i < OrchestrationCount; i++)
{
    // Create a task for each orchestration schedule but don't await it yet
    scheduleTasks.Add(client.ScheduleNewOrchestrationInstanceAsync(
        "GreetingOrchestration", 
        $"{name}_{i}"));
}

logger.LogInformation("Created {Count} scheduling tasks, waiting for all to complete...", OrchestrationCount);

// Wait for all scheduling tasks to complete simultaneously
string[] results = await Task.WhenAll(scheduleTasks);
instanceIds.AddRange(results);

// Log the completion of scheduling
stopwatch.Stop();
logger.LogInformation("Scheduled {Count} orchestrations concurrently in {ElapsedMs}ms", 
    OrchestrationCount, stopwatch.ElapsedMilliseconds);

// Wait for orchestrations to complete
logger.LogInformation("Waiting for orchestrations to complete...");

// Create a cancellation token source with extended timeout for multiple orchestrations
using CancellationTokenSource cts = new CancellationTokenSource(TimeSpan.FromMinutes(5));

// Track completion stats
int completed = 0;
int failed = 0;
stopwatch.Restart();

// Create tasks for waiting for all orchestrations to complete
var waitTasks = new List<Task<OrchestrationMetadata>>(OrchestrationCount);
foreach (string id in instanceIds)
{
    waitTasks.Add(client.WaitForInstanceCompletionAsync(
        id,
        getInputsAndOutputs: false,  // Set to false to reduce overhead
        cts.Token));
}

// Process completion results as they arrive
while (waitTasks.Count > 0)
{
    // Wait for any task to complete
    Task<OrchestrationMetadata> completedTask = await Task.WhenAny(waitTasks);
    waitTasks.Remove(completedTask);
    
    try
    {
        OrchestrationMetadata instance = await completedTask;
        
        if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
        {
            completed++;
        }
        else if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Failed)
        {
            failed++;
            logger.LogError("Orchestration {Id} failed: {ErrorMessage}", 
                instance.InstanceId, instance.FailureDetails?.ErrorMessage);
        }
        
        // Log progress every 200 instances
        if ((completed + failed) % 200 == 0)
        {
            logger.LogInformation("Progress: {Completed} completed, {Failed} failed, {Remaining} remaining", 
                completed, failed, waitTasks.Count);
        }
    }
    catch (OperationCanceledException)
    {
        logger.LogWarning("Timeout waiting for orchestration to complete");
    }
}

stopwatch.Stop();
logger.LogInformation("Completed {SuccessCount}/{TotalCount} orchestrations in {ElapsedMs}ms", 
    completed, OrchestrationCount, stopwatch.ElapsedMilliseconds);
logger.LogInformation("Success rate: {SuccessRate}%", (double)completed / OrchestrationCount * 100);

// Keep the client running in container environments or exit gracefully in interactive environments
logger.LogInformation("Task completed successfully.");
if (Environment.UserInteractive && !Console.IsInputRedirected)
{
    logger.LogInformation("Press any key to exit...");
    Console.ReadKey();
}
else
{
    // In non-interactive environments like containers, wait for a signal to shut down
    logger.LogInformation("Running in non-interactive mode. Service will stay alive.");
    
    // Create a simple way to handle shutdown signals
    var waitForShutdown = new TaskCompletionSource<bool>();
    Console.CancelKeyPress += (sender, e) => {
        e.Cancel = true;
        waitForShutdown.TrySetResult(true);
    };
    
    // Wait indefinitely or until shutdown signal
    await waitForShutdown.Task;
}