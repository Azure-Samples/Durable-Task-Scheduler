using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
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
logger.LogInformation("This worker implements a news article generator workflow with multiple specialized agents");

// Create loggers for each service and activity
var researchAgentServiceLogger = loggerFactory.CreateLogger<ResearchAgentService>();
var contentGenerationAgentServiceLogger = loggerFactory.CreateLogger<ContentGenerationAgentService>();
var imageGenerationAgentServiceLogger = loggerFactory.CreateLogger<ImageGenerationAgentService>();
var researchTopicActivityLogger = loggerFactory.CreateLogger<ResearchTopicActivity>();
var createArticleActivityLogger = loggerFactory.CreateLogger<CreateArticleActivity>();
var generateImagesActivityLogger = loggerFactory.CreateLogger<GenerateImagesActivity>();
var assembleFinalArticleActivityLogger = loggerFactory.CreateLogger<AssembleFinalArticleActivity>();
var orchestrationLogger = loggerFactory.CreateLogger<ContentCreationOrchestration>();

// Create service instances
var researchAgentService = new ResearchAgentService(researchAgentServiceLogger);
var contentGenerationAgentService = new ContentGenerationAgentService(contentGenerationAgentServiceLogger);
var imageGenerationAgentService = new ImageGenerationAgentService(imageGenerationAgentServiceLogger);

// Create activity instances
var researchTopicActivity = new ResearchTopicActivity(researchAgentService, researchTopicActivityLogger);
var createArticleActivity = new CreateArticleActivity(contentGenerationAgentService, createArticleActivityLogger);
var generateImagesActivity = new GenerateImagesActivity(imageGenerationAgentService, generateImagesActivityLogger);
var assembleFinalArticleActivity = new AssembleFinalArticleActivity(assembleFinalArticleActivityLogger);

// Create orchestration instance
var contentCreationOrchestration = new ContentCreationOrchestration(orchestrationLogger);

// Configure services
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        // Register the orchestration
        registry.AddOrchestratorFunc<ContentCreationRequest, ContentWorkflowResult>(
            "ContentCreationOrchestration", 
            (ctx, input) => contentCreationOrchestration.RunAsync(ctx, input));
        
        // Register the activities
        registry.AddActivityFunc<string, ResearchData>(
            "ResearchTopicActivity", 
            (ctx, input) => researchTopicActivity.RunAsync(input));
            
        registry.AddActivityFunc<(string topic, ResearchData researchData), string>(
            "CreateArticleActivity", 
            (ctx, input) => createArticleActivity.RunAsync(input));
            
        registry.AddActivityFunc<(string topic, string articleContent), List<GeneratedImage>>(
            "GenerateImagesActivity", 
            (ctx, input) => generateImagesActivity.RunAsync(input));
            
        registry.AddActivityFunc<(string articleContent, List<GeneratedImage> images), ArticleResult>(
            "AssembleFinalArticleActivity", 
            (ctx, input) => assembleFinalArticleActivity.RunAsync(input));
    })
    .UseDurableTaskScheduler(connectionString);

// Build the host
IHost host = builder.Build();

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
