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
logger.LogInformation("Starting Autoscaling in Azure Container Apps - Greeting Client");

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
logger.LogInformation("Starting perpetual orchestration scheduler - 5 orchestrations every 5 seconds");

// Set up orchestration batch parameters
const int BatchSize = 5;           // Number of orchestrations per batch
const int IntervalSeconds = 5;     // Time between batches in seconds
int batchNumber = 0;               // Track which batch we're on
var completedOrchestrations = 0;   // Track total completed orchestrations
var failedOrchestrations = 0;      // Track total failed orchestrations

// Create a cancellation token source that will be used to signal shutdown
using var appShutdownCts = new CancellationTokenSource();

// Register console cancellation to trigger graceful shutdown
Console.CancelKeyPress += (sender, e) => {
    e.Cancel = true;
    logger.LogInformation("Shutdown signal received. Completing current batch and exiting...");
    appShutdownCts.Cancel();
};

// Start the perpetual scheduling loop
_ = Task.Run(async () => {
    try 
    {
        while (!appShutdownCts.Token.IsCancellationRequested)
        {
            batchNumber++;
            logger.LogInformation("Scheduling batch #{BatchNumber} ({BatchSize} orchestrations)", batchNumber, BatchSize);
            
            // Create a stopwatch to measure batch performance
            var batchStopwatch = Stopwatch.StartNew();
            
            // Schedule a batch of orchestrations concurrently
            var scheduleTasks = new List<Task<string>>(BatchSize);
            for (int i = 0; i < BatchSize; i++)
            {
                // Create a unique instance ID based on timestamp and index
                string instanceName = $"{name}_batch{batchNumber}_{i}";
                
                // Add scheduling task to batch
                scheduleTasks.Add(client.ScheduleNewOrchestrationInstanceAsync(
                    "GreetingOrchestration", 
                    instanceName));
            }
            
            try
            {
                // Wait for all orchestrations in this batch to be scheduled
                string[] batchInstanceIds = await Task.WhenAll(scheduleTasks);
                batchStopwatch.Stop();
                
                logger.LogInformation("Batch #{BatchNumber} scheduled in {ElapsedMs}ms", 
                    batchNumber, batchStopwatch.ElapsedMilliseconds);
                
                // Start monitoring batch completion in the background
                _ = MonitorBatchCompletionAsync(batchInstanceIds, batchNumber);
            }
            catch (Exception ex)
            {
                logger.LogError(ex, "Error scheduling batch #{BatchNumber}", batchNumber);
            }
            
            // Wait for the configured interval before scheduling the next batch
            try
            {
                await Task.Delay(TimeSpan.FromSeconds(IntervalSeconds), appShutdownCts.Token);
            }
            catch (TaskCanceledException)
            {
                // This is expected when shutdown is requested
                break;
            }
        }
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error in scheduling loop");
    }
    finally
    {
        logger.LogInformation("Scheduling loop terminated");
    }
});

// Method to monitor batch completion
async Task MonitorBatchCompletionAsync(string[] batchInstanceIds, int batchNum)
{
    try
    {
        // Create tasks for waiting for orchestrations in this batch to complete
        var batchWaitTasks = new List<Task<OrchestrationMetadata>>(batchInstanceIds.Length);
        foreach (string id in batchInstanceIds)
        {
            batchWaitTasks.Add(client.WaitForInstanceCompletionAsync(id, getInputsAndOutputs: false, CancellationToken.None));
        }
        
        // Process completion results as they arrive
        int batchCompleted = 0;
        int batchFailed = 0;
        
        while (batchWaitTasks.Count > 0)
        {
            // Wait for any orchestration to complete
            Task<OrchestrationMetadata> completedTask = await Task.WhenAny(batchWaitTasks);
            batchWaitTasks.Remove(completedTask);
            
            try
            {
                OrchestrationMetadata instance = await completedTask;
                
                if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
                {
                    batchCompleted++;
                    Interlocked.Increment(ref completedOrchestrations);
                }
                else if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Failed)
                {
                    batchFailed++;
                    Interlocked.Increment(ref failedOrchestrations);
                    logger.LogError("Orchestration {Id} failed: {ErrorMessage}", 
                        instance.InstanceId, instance.FailureDetails?.ErrorMessage);
                }
            }
            catch (Exception ex)
            {
                logger.LogError(ex, "Error processing orchestration completion");
            }
        }
        
        // Log batch completion stats
        logger.LogInformation("Batch #{BatchNumber} completed: {Completed} succeeded, {Failed} failed", 
            batchNum, batchCompleted, batchFailed);
            
        // Log overall stats periodically (every 10 batches)
        if (batchNum % 10 == 0)
        {
            logger.LogInformation("OVERALL STATS: {Completed} completed, {Failed} failed, {Total} total orchestrations", 
                completedOrchestrations, failedOrchestrations, completedOrchestrations + failedOrchestrations);
        }
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error monitoring batch #{BatchNumber}", batchNum);
    }
}

// Keep the app running until shutdown is requested
logger.LogInformation("Perpetual orchestration scheduler running. Press Ctrl+C to exit...");
await Task.Delay(Timeout.Infinite, appShutdownCts.Token).ContinueWith(_ => Task.CompletedTask);

logger.LogInformation("Application shutdown complete");