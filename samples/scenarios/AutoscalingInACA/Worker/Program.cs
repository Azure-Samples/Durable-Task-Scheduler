using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using AutoscalingInACA;

// Configure the host builder
HostApplicationBuilder builder = Host.CreateApplicationBuilder();

// Configure logging
builder.Logging.AddConsole();
builder.Logging.SetMinimumLevel(LogLevel.Information);

// Build a logger for startup configuration
using ILoggerFactory loggerFactory = LoggerFactory.Create(loggingBuilder =>
{
    loggingBuilder.AddConsole();
    loggingBuilder.SetMinimumLevel(LogLevel.Information);
});
ILogger<Program> logger = loggerFactory.CreateLogger<Program>();

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
logger.LogInformation("This worker implements a simple greeting workflow with 3 chained activities");

// Configure services
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddAllGeneratedTasks(); // This will add the tasks decorated with [DurableTask]
    })
    .UseDurableTaskScheduler(connectionString);

// Build the host
IHost host = builder.Build();

// Get the logger from the service provider for the rest of the program
logger = host.Services.GetRequiredService<ILogger<Program>>();

logger.LogInformation("Starting Autoscaling in Azure Container Apps - Greeting Worker");

// Start the host
await host.StartAsync();

logger.LogInformation("Worker started and waiting for tasks...");

// Wait indefinitely in environments without interactive console,
// or until a key is pressed in interactive environments
if (Environment.UserInteractive && !Console.IsInputRedirected)
{
    logger.LogInformation("Press any key to stop...");
    Console.ReadKey();
}
else
{
    // In non-interactive environments (like containers), wait indefinitely
    await Task.Delay(Timeout.InfiniteTimeSpan);
}

// Stop the host
await host.StopAsync();
