using System;
using System.Threading;
using System.Threading.Tasks;
using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using AgentChainingSample.Services;
using AgentChainingSample.Activities;
using AgentChainingSample.Orchestrations;
using AgentChainingSample.Shared.Models;

// Configure the host builder
HostApplicationBuilder builder = Host.CreateApplicationBuilder();

// Configure logging
builder.Logging.AddConsole();
builder.Logging.SetMinimumLevel(LogLevel.Information);

// Add configuration to services
builder.Services.AddSingleton(builder.Configuration);

// Add HttpClient factory for proper management of HTTP connections
builder.Services.AddHttpClient();

// Register named HttpClient for DALL-E with appropriate timeouts
builder.Services.AddHttpClient("DallEClient", client =>
{
    client.Timeout = TimeSpan.FromSeconds(120); // Increase timeout for image generation
});

// Register services with DI
builder.Services.AddTransient<ResearchAgentService>();
builder.Services.AddTransient<ContentGenerationAgentService>();
builder.Services.AddTransient<ImageGenerationAgentService>();

// Activities with [DurableTask] attribute are auto-registered via AddAllGeneratedTasks()
// No need to manually register them here

// Get connection string from configuration with fallback to default local emulator connection
string connectionString = builder.Configuration["DTS_CONNECTION_STRING"] ?? 
                         "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

// Configure services
// Register tasks with DI
builder.Services.AddDurableTaskWorker(builder =>
{
    builder.AddTasks(registry => 
    {
        // Auto-register all tasks marked with [DurableTask] attribute
        registry.AddAllGeneratedTasks();
    });
    builder.UseDurableTaskScheduler(connectionString);
});

// Build the host
IHost host = builder.Build();

// Get a proper logger from the service provider
var logger = host.Services.GetRequiredService<ILogger<Program>>();
logger.LogInformation("Connection string: {ConnectionString}", connectionString);
logger.LogInformation("This worker implements a news article generator workflow with multiple specialized agents");
logger.LogInformation("Starting Agent Chaining Sample Worker");

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
