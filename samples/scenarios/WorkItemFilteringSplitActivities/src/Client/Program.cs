using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;

// Configure logging
using ILoggerFactory loggerFactory = LoggerFactory.Create(builder =>
{
    builder.AddSimpleConsole(options =>
    {
        options.SingleLine = true;
        options.TimestampFormat = "HH:mm:ss ";
    });
    builder.SetMinimumLevel(LogLevel.Information);
});

ILogger logger = loggerFactory.CreateLogger("Client");

logger.LogInformation("=== Work Item Filtering Demo — Client ===");

// Build connection string
string endpoint = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHubName = Environment.GetEnvironmentVariable("TASKHUB") ?? "default";
string? managedIdentityClientId = Environment.GetEnvironmentVariable("AZURE_MANAGED_IDENTITY_CLIENT_ID");
bool isLocalEmulator = endpoint == "http://localhost:8080";

string connectionString = isLocalEmulator
    ? $"Endpoint={endpoint};TaskHub={taskHubName};Authentication=None"
    : !string.IsNullOrEmpty(managedIdentityClientId)
        ? $"Endpoint={endpoint};TaskHub={taskHubName};Authentication=ManagedIdentity;ClientID={managedIdentityClientId}"
        : $"Endpoint={endpoint};TaskHub={taskHubName};Authentication=DefaultAzure";

logger.LogInformation("Connection: {ConnectionString}", connectionString);

// Create the Durable Task client
ServiceCollection services = new();
services.AddLogging(lb =>
{
    lb.AddSimpleConsole(o => { o.SingleLine = true; o.TimestampFormat = "HH:mm:ss "; });
    lb.SetMinimumLevel(LogLevel.Information);
});
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

await using ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

// Run in a loop: schedule a batch of orchestrations every 30 seconds for 15 minutes
const int orchestrationsPerBatch = 3;
TimeSpan interval = TimeSpan.FromSeconds(30);
TimeSpan totalDuration = TimeSpan.FromMinutes(10);
DateTime deadline = DateTime.UtcNow + totalDuration;

int totalCompleted = 0;
int totalFailed = 0;
int batchNumber = 0;

logger.LogInformation("Will schedule {Count} orchestrations every {Interval}s for {Duration} minutes.",
    orchestrationsPerBatch, interval.TotalSeconds, totalDuration.TotalMinutes);
logger.LogInformation("(Make sure the Orchestrator, Validator, and Shipper workers are all running)\n");

while (DateTime.UtcNow < deadline)
{
    batchNumber++;
    logger.LogInformation("--- Batch #{Batch} at {Time:HH:mm:ss} ---", batchNumber, DateTime.UtcNow);

    var instanceIds = new List<string>();
    for (int i = 1; i <= orchestrationsPerBatch; i++)
    {
        string orderId = $"ORD-B{batchNumber:D3}-{i:D3}";
        logger.LogInformation("Scheduling orchestration with orderId='{OrderId}'...", orderId);

        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            "OrderProcessingOrchestration",
            orderId);

        instanceIds.Add(instanceId);
        logger.LogInformation("  -> Scheduled with InstanceId={InstanceId}", instanceId);
    }

    // Wait for all orchestrations in this batch to complete
    int batchCompleted = 0;
    int batchFailed = 0;

    foreach (string id in instanceIds)
    {
        try
        {
            OrchestrationMetadata result = await client.WaitForInstanceCompletionAsync(
                id, getInputsAndOutputs: true, CancellationToken.None);

            if (result.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
            {
                batchCompleted++;
                logger.LogInformation(
                    "COMPLETED | InstanceId={InstanceId} | Output: {Output}",
                    result.InstanceId, result.ReadOutputAs<string>());
            }
            else
            {
                batchFailed++;
                logger.LogError(
                    "FAILED    | InstanceId={InstanceId} | Status={Status} | Error: {Error}",
                    result.InstanceId, result.RuntimeStatus, result.FailureDetails?.ErrorMessage);
            }
        }
        catch (Exception ex)
        {
            batchFailed++;
            logger.LogError(ex, "Error waiting for orchestration {Id}", id);
        }
    }

    totalCompleted += batchCompleted;
    totalFailed += batchFailed;

    logger.LogInformation("Batch #{Batch} results: {Completed} completed, {Failed} failed",
        batchNumber, batchCompleted, batchFailed);

    // Wait for the next interval (unless we've passed the deadline)
    if (DateTime.UtcNow < deadline)
    {
        TimeSpan remaining = deadline - DateTime.UtcNow;
        TimeSpan waitTime = remaining < interval ? remaining : interval;
        logger.LogInformation("Next batch in {Seconds:F0}s (deadline in {Remaining:F1} min)\n",
            waitTime.TotalSeconds, remaining.TotalMinutes);
        await Task.Delay(waitTime);
    }
}

logger.LogInformation("\n=== FINAL RESULTS: {Completed} completed, {Failed} failed across {Batches} batches ===",
    totalCompleted, totalFailed, batchNumber);

// Keep the process alive so Container Apps doesn't mark it as failed
logger.LogInformation("Demo complete. Staying alive — press Ctrl+C to exit.");
await Task.Delay(Timeout.Infinite, CancellationToken.None);
