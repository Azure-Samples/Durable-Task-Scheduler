// filepath: /Users/nickgreenfield1/workspace/Durable-Task-Scheduler/samples/durabletask-sdks/dotnet/HumanInteraction/Client/Program.cs
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
logger.LogInformation("Starting Human Interaction Pattern - Approval Client");

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
    // For Azure, use DefaultAzureCredential
    connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=DefaultAzureCredential";
    logger.LogInformation("Using Azure endpoint with DefaultAzureCredential");
}

logger.LogInformation("Using endpoint: {Endpoint}", endpoint);
logger.LogInformation("Using task hub: {TaskHubName}", taskHubName);
logger.LogInformation("Host address: {HostAddress}", hostAddress);
logger.LogInformation("Connection string: {ConnectionString}", connectionString);
logger.LogInformation("This client creates an approval request and handles the response");

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

// Create an approval request
var requestId = Guid.NewGuid().ToString();
var requester = "Console User";
var item = "Vacation Request";
var timeoutHours = 1.0; // Short timeout for demonstration

// Prepare input for the orchestration
var input = new
{
    RequestId = requestId,
    Requester = requester,
    Item = item,
    TimeoutHours = timeoutHours
};

logger.LogInformation("Creating new approval request with ID: {RequestId}", requestId);
logger.LogInformation("Request details: Requester: {Requester}, Item: {Item}, Timeout: {TimeoutHours}h",
    requester, item, timeoutHours);

// Schedule the orchestration
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "ApprovalOrchestration",
    input,
    new StartOrchestrationOptions(requestId)); // Use requestId as instanceId for simplicity

logger.LogInformation("Started orchestration with ID: {InstanceId}", instanceId);

// Check initial status
logger.LogInformation("Checking initial status...");
OrchestrationMetadata? initialStatus = await client.GetInstanceAsync(instanceId, true);
PrintStatus(initialStatus);

// Wait for user decision
Console.WriteLine("\nPress Enter to approve the request, or type 'reject' and press Enter to reject: ");
string userInput = Console.ReadLine() ?? "";
bool isApproved = !userInput.Trim().Equals("reject", StringComparison.OrdinalIgnoreCase);

// Create an ApprovalResponseData object with the expected structure
var approvalResponse = new HumanInteraction.Client.ApprovalResponseData
{
    IsApproved = isApproved,
    Approver = "Console User",
    Comments = "Response from console application",
    ResponseTime = DateTime.UtcNow.ToString("o")
};

// Send approval response as an external event
logger.LogInformation("Submitting your response ({Response})...", isApproved ? "Approved" : "Rejected");
await client.RaiseEventAsync(
    instanceId,
    "approval_response",
    approvalResponse);

// Wait for final status
logger.LogInformation("Waiting for final status...");

// Create a cancellation token source with timeout
using CancellationTokenSource cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));

// Wait for the orchestration to complete or check a few times
OrchestrationMetadata? finalStatus = null;
for (int i = 0; i < 5; i++)
{
    await Task.Delay(TimeSpan.FromSeconds(2));

    finalStatus = await client.GetInstanceAsync(instanceId, true);

    if (finalStatus != null &&
        (finalStatus.RuntimeStatus == OrchestrationRuntimeStatus.Completed ||
         finalStatus.RuntimeStatus == OrchestrationRuntimeStatus.Failed))
    {
        break;
    }
}

// Print final status
logger.LogInformation("Final status:");
PrintStatus(finalStatus);

logger.LogInformation("Sample completed.");

// Helper method to print status details
static void PrintStatus(OrchestrationMetadata? status)
{
    if (status == null)
    {
        Console.WriteLine("  Status: Not found");
        return;
    }

    Console.WriteLine($"  Status: {status.RuntimeStatus}");

    // Print custom status if available
    if (!string.IsNullOrEmpty(status.SerializedCustomStatus))
    {
        Console.WriteLine("  Details:");
        var customStatus = JsonSerializer.Serialize(JsonDocument.Parse(status.SerializedCustomStatus), new JsonSerializerOptions
        {
            WriteIndented = true
        });
        Console.WriteLine($"{customStatus}");
    }

    // Print output if available
    if (status.RuntimeStatus == OrchestrationRuntimeStatus.Completed && status.SerializedOutput != null)
    {
        var output = JsonDocument.Parse(status.SerializedOutput);
        Console.WriteLine("  Output:");
        Console.WriteLine(JsonSerializer.Serialize(output, new JsonSerializerOptions
        {
            WriteIndented = true
        }));
    }

    // Print failure details if available
    if (status.RuntimeStatus == OrchestrationRuntimeStatus.Failed && status.FailureDetails != null)
    {
        Console.WriteLine($"  Error: {status.FailureDetails.ErrorMessage}");
    }
}
