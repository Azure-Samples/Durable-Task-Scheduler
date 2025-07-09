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
// Force rebuild timestamp: 2025-01-06 10:26
using AgentChainingSample.Activities;
using AgentChainingSample.Orchestrations;
using AgentChainingSample.Services;
using AgentChainingSample.Worker.Models;

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

// Register services with DI as singletons
// These services perform initialization work that doesn't need to be repeated for each activity
// Thread safety for initialization is handled by the BaseAgentService implementation
builder.Services.AddSingleton<ResearchAgentService>();
builder.Services.AddSingleton<ContentGenerationAgentService>();
builder.Services.AddSingleton<ImageGenerationAgentService>();

// Activities with [DurableTask] attribute are auto-registered via AddAllGeneratedTasks()
// No need to manually register them here

// Get connection string from configuration with fallback to default local emulator connection
string connectionString = builder.Configuration["ENDPOINT"] ??
                         builder.Configuration["DTS_CONNECTION_STRING"] ?? 
                         "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

// If we have the endpoint but not a full connection string, construct it
if (connectionString.StartsWith("Endpoint=") && !connectionString.Contains("TaskHub="))
{
    string taskHub = builder.Configuration["TASKHUB"] ?? builder.Configuration["TASKHUB_NAME"] ?? "default";
    string clientId = builder.Configuration["AZURE_MANAGED_IDENTITY_CLIENT_ID"] ?? "";
    
    if (!string.IsNullOrEmpty(clientId))
    {
        connectionString = $"{connectionString};Authentication=ManagedIdentity;ClientID={clientId};TaskHub={taskHub}";
    }
    else
    {
        connectionString = $"{connectionString};TaskHub={taskHub}";
    }
}

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
// Log the constructed connection string with sensitive info redacted
string managedIdentityClientId = builder.Configuration["AZURE_MANAGED_IDENTITY_CLIENT_ID"] ?? "";
string logConnectionString = !string.IsNullOrEmpty(managedIdentityClientId) ? 
    connectionString.Replace(managedIdentityClientId, "[REDACTED]") : 
    connectionString;
logger.LogInformation("Connection string: {ConnectionString}", logConnectionString);
logger.LogInformation("TaskHub: {TaskHub}", builder.Configuration["TASKHUB"] ?? builder.Configuration["TASKHUB_NAME"] ?? "default");
logger.LogInformation("This worker implements a news article generator workflow with multiple specialized agents");

// Log OpenAI configuration
logger.LogInformation("Azure OpenAI Endpoint: {Endpoint}", builder.Configuration["AZURE_OPENAI_ENDPOINT"] ?? "Not set");
logger.LogInformation("Azure OpenAI Deployment: {Deployment}", builder.Configuration["OPENAI_DEPLOYMENT_NAME"] ?? "gpt-4 (default)");
logger.LogInformation("DALL-E Endpoint: {DalleEndpoint}", !string.IsNullOrEmpty(builder.Configuration["DALLE_ENDPOINT"]) ? 
    "Configured" : "Not set - will use placeholder images");
logger.LogInformation("Agent Connection String: {AgentConnectionString}", !string.IsNullOrEmpty(builder.Configuration["AGENT_CONNECTION_STRING"]) ? 
    "Configured" : "Not set - required for agent functionality");

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