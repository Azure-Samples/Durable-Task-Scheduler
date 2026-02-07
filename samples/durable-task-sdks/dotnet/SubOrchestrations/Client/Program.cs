using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;

using ILoggerFactory loggerFactory = LoggerFactory.Create(builder =>
{
    builder.AddConsole();
    builder.SetMinimumLevel(LogLevel.Information);
});

ILogger<Program> logger = loggerFactory.CreateLogger<Program>();

string connectionString = Environment.GetEnvironmentVariable("DTS_CONNECTION_STRING")
    ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

logger.LogInformation("Connection string: {ConnectionString}", connectionString);

ServiceCollection services = new ServiceCollection();
services.AddLogging(builder => builder.AddConsole());
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

Console.WriteLine("Starting order processing orchestration...");

// Create a sample order with multiple line items
var order = new
{
    OrderId = $"ORD-{DateTime.UtcNow:yyyyMMdd}-{Random.Shared.Next(1000, 9999)}",
    Items = new[]
    {
        new { ProductName = "Laptop", Quantity = 1, UnitPrice = 999.99m },
        new { ProductName = "Mouse", Quantity = 2, UnitPrice = 29.99m },
        new { ProductName = "Keyboard", Quantity = 1, UnitPrice = 79.99m },
    }
};

string instanceId = await client.ScheduleNewOrchestrationInstanceAsync("OrderOrchestration", order);
Console.WriteLine($"Order '{order.OrderId}' submitted. Instance: {instanceId}");
Console.WriteLine("Waiting for completion...");

OrchestrationMetadata metadata = await client.WaitForInstanceCompletionAsync(
    instanceId, getInputsAndOutputs: true, CancellationToken.None);
Console.WriteLine($"Status: {metadata.RuntimeStatus}");
Console.WriteLine($"Result: {metadata.SerializedOutput}");
