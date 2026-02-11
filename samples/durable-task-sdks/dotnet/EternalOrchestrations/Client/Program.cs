using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;

using ILoggerFactory loggerFactory = LoggerFactory.Create(builder =>
{
    builder.AddSimpleConsole(options =>
    {
        options.SingleLine = true;
        options.UseUtcTimestamp = true;
    });
});

ILogger logger = loggerFactory.CreateLogger("Client");

string connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

var services = new ServiceCollection();
services.AddLogging(builder => builder.AddConsole());
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

Console.WriteLine("Starting eternal cleanup orchestration...");
Console.WriteLine("This orchestration runs indefinitely. Use the dashboard to monitor it.");
Console.WriteLine("Press Ctrl+C to exit this client (the orchestration continues running).\n");

string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "EternalCleanupOrchestration",
    new { Iteration = 0, IntervalSeconds = 10 });

Console.WriteLine($"Orchestration started: {instanceId}");
Console.WriteLine($"Dashboard: http://localhost:8082\n");

// Monitor for a while to show it working
for (int i = 0; i < 5; i++)
{
    await Task.Delay(TimeSpan.FromSeconds(12));
    var metadata = await client.GetInstanceAsync(instanceId);
    Console.WriteLine($"Status: {metadata?.RuntimeStatus} (checking iteration progress...)");
}

Console.WriteLine("\nEternal orchestration is still running. Terminate it from the dashboard or with:");
Console.WriteLine($"  // client.TerminateInstanceAsync(\"{instanceId}\", \"Manual termination\");");
