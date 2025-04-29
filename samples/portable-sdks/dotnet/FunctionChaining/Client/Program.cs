using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using System.Text.Json;

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
logger.LogInformation("Starting greeting orchestration for name: {Name}", name);

// Schedule the orchestration
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "GreetingOrchestration", 
    name);

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
    var result = instance.ReadOutputAs<string>();
    logger.LogInformation("Greeting result: {Result}", result);
}
else if (instance.RuntimeStatus == OrchestrationRuntimeStatus.Failed)
{
    logger.LogError("Orchestration failed: {ErrorMessage}", instance.FailureDetails?.ErrorMessage);
}

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