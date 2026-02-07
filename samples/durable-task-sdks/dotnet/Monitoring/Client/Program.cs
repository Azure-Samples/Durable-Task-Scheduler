using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

var builder = Host.CreateApplicationBuilder(args);

builder.Services.AddDurableTaskClient(options =>
{
    options.UseDurableTaskScheduler("Endpoint=http://localhost:8080;TaskHub=default;Authentication=None");
});

builder.Logging.AddSimpleConsole(options =>
{
    options.SingleLine = true;
    options.UseUtcTimestamp = true;
});

var host = builder.Build();
await host.StartAsync();

var client = host.Services.GetRequiredService<DurableTaskClient>();

string jobId = $"job-{Guid.NewGuid():N}"[..12];
Console.WriteLine($"Starting monitor orchestration for job '{jobId}'...");

string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "MonitorOrchestration",
    new { JobId = jobId, PollingIntervalSeconds = 3, MaxPolls = 10, CurrentPoll = 0 });

Console.WriteLine($"Orchestration started: {instanceId}");
Console.WriteLine("Waiting for completion...");

var metadata = await client.WaitForInstanceCompletionAsync(instanceId, getInputsAndOutputs: true);
Console.WriteLine($"Result: {metadata?.ReadOutputAs<string>()}");

await host.StopAsync();
