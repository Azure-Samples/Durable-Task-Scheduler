using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;

string endpoint = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHub = Environment.GetEnvironmentVariable("TASKHUB") ?? "default";
string connectionString = endpoint.Contains("localhost")
    ? $"Endpoint={endpoint};TaskHub={taskHub};Authentication=None"
    : $"Endpoint={endpoint};TaskHub={taskHub};Authentication=DefaultAzure";

var services = new ServiceCollection();
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

await using ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

Console.WriteLine("Scheduling order processing orchestration...");
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "OrderProcessingOrchestration",
    input: "Order-12345");

Console.WriteLine($"Started orchestration: {instanceId}");
Console.WriteLine("Waiting for completion...");

var result = await client.WaitForInstanceCompletionAsync(instanceId, getInputsAndOutputs: true);
Console.WriteLine($"Status: {result.RuntimeStatus}");
Console.WriteLine($"Result: {result.ReadOutputAs<string>()}");
Console.WriteLine();
Console.WriteLine("View traces in Jaeger UI: http://localhost:16686");
Console.WriteLine("View orchestration in DTS Dashboard: http://localhost:8082");
