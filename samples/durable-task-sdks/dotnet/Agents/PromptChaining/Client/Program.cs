using Azure.Identity;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using AgentChainingSample.Shared.Models;
using System.Text.Json;
using System.Threading.Tasks;
using System.Diagnostics;

// Configure configuration
var configuration = new ConfigurationBuilder()
    .AddEnvironmentVariables()
    .Build();

// Configure logging
using ILoggerFactory loggerFactory = LoggerFactory.Create(builder =>
{
    builder.AddConsole();
    builder.SetMinimumLevel(LogLevel.Information);
});

ILogger<Program> logger = loggerFactory.CreateLogger<Program>();
logger.LogInformation("Starting Agent Chaining Sample - Content Creation Client");

// Get connection string from configuration with fallback to default local emulator connection
string connectionString = configuration["DTS_CONNECTION_STRING"] ?? 
                         "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

// Determine if we're connecting to the local emulator
bool isLocalEmulator = connectionString.Contains("localhost");

// Enable HTTP/2 cleartext support for emulator
if (isLocalEmulator)
{
    AppContext.SetSwitch("System.Net.Http.SocketsHttpHandler.Http2UnencryptedSupport", true);
    logger.LogInformation("Using local emulator");
}
else
{
    logger.LogInformation("Using Azure endpoint with DefaultAzure authentication");
}

logger.LogInformation("Connection string: {ConnectionString}", connectionString);
logger.LogInformation("This sample implements a news article generator workflow with multiple specialized agents");

// Create the client using DI service provider
ServiceCollection services = new ServiceCollection();
services.AddLogging(builder => builder.AddConsole());

// Register the client, which can be used to start orchestrations
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

try
{
    // Ask for the news topic
    Console.WriteLine("\nEnter a news topic to research and generate an article (or type 'exit' to quit):");
    string? topic = Console.ReadLine();
    
    while (!string.IsNullOrWhiteSpace(topic) && topic.ToLower() != "exit")
    {
        // Create the request
        ContentCreationRequest request = new ContentCreationRequest
        {
            Topic = topic,
            RequestId = Guid.NewGuid().ToString()
        };
        
        // Start the orchestration
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            "ContentCreationOrchestration", 
            request);
        
        logger.LogInformation("Started orchestration with ID: {InstanceId}", instanceId);
        
        // Wait for the orchestration to complete with timeout
        logger.LogInformation("Waiting for orchestration to complete...");
        using var timeoutCts = new CancellationTokenSource(TimeSpan.FromMinutes(5));
        OrchestrationMetadata metadata = await client.WaitForInstanceCompletionAsync(
            instanceId,
            getInputsAndOutputs: true,
            timeoutCts.Token);
        
        if (metadata.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
        {
            ContentWorkflowResult? result = metadata.ReadOutputAs<ContentWorkflowResult>();
            
            Console.ForegroundColor = ConsoleColor.Green;
            Console.WriteLine("\n===== NEWS ARTICLE GENERATION COMPLETED =====");
            Console.ResetColor();
            
            Console.WriteLine($"\nTopic: {result?.Topic}");
            
            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine("\n----- RESEARCH SUMMARY -----");
            Console.ResetColor();
            Console.WriteLine(result?.ResearchData.Summary);
            
            Console.WriteLine("\nSources Found:");
            foreach (var source in result?.ResearchData.Sources ?? new List<ResearchSource>())
            {
                Console.WriteLine($"- {source.Title}: {source.Url}");
            }
            
            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine("\n----- ARTICLE CONTENT -----");
            Console.ResetColor();
            Console.WriteLine(result?.ArticleContent);
            
            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine("\n----- GENERATED IMAGES -----");
            Console.ResetColor();
            foreach (var image in result?.GeneratedImages ?? new List<GeneratedImage>())
            {
                Console.WriteLine($"- {image.Description}");
                Console.WriteLine($"  Caption: {image.Caption}");
                Console.WriteLine($"  DALL-E Prompt: {image.Prompt}");
                Console.WriteLine();
            }
            
            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine("\n----- COMPLETE ARTICLE WITH IMAGES -----");
            Console.ResetColor();
            
            if (!string.IsNullOrEmpty(result?.ArticleBlobUrl))
            {
                Console.WriteLine($"Article is available online at: {result?.ArticleBlobUrl}");
                Console.WriteLine($"Local HTML file saved at: {result?.ArticleFilePath}");
                Console.WriteLine();
            }
            
            Console.WriteLine(result?.FinalArticle);
            
            // Ask for another topic
            Console.WriteLine("\nEnter another news topic to research (or type 'exit' to quit):");
            topic = Console.ReadLine();
        }
        else
        {
            Console.WriteLine($"Orchestration ended with status: {metadata.RuntimeStatus}");
            break;
        }
    }
}
catch (Exception ex)
{
    logger.LogError(ex, "Error in client application");
}

logger.LogInformation("Client application stopped");
