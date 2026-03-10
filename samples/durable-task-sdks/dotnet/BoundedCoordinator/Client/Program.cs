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

ServiceCollection services = new();
services.AddLogging(builder => builder.AddConsole());
services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

ServiceProvider serviceProvider = services.BuildServiceProvider();
DurableTaskClient client = serviceProvider.GetRequiredService<DurableTaskClient>();

string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "CoordinatorOrchestration",
    input: default(object));

logger.LogInformation("Started CoordinatorOrchestration, instanceId={InstanceId}", instanceId);

var metadata = await client.WaitForInstanceCompletionAsync(instanceId, getInputsAndOutputs: true);

logger.LogInformation("Completed with status={Status}", metadata.RuntimeStatus);
