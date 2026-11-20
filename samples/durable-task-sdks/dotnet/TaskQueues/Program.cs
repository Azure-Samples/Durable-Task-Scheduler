using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using TaskQueues;

HostApplicationBuilder builder = Host.CreateApplicationBuilder(args);
string connectionString = Environment.GetEnvironmentVariable("DTS_CONNECTION_STRING")
    ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

builder.Services.AddDurableTaskClient(client => client.UseDurableTaskScheduler(connectionString));
builder.Services.AddDurableTaskWorker(worker =>
{
    worker.AddTasks(tasks =>
    {
        tasks.AddOrchestrator<ProcessOrder>();
        tasks.AddOrchestrator<ValidateOrder>();
        tasks.AddActivity<RecordOrder>();
        tasks.AddActivity<CallPartner>();
    });
    worker.UseDurableTaskScheduler(connectionString);
    worker.UseWorkItemFilters(null);
});

using IHost host = builder.Build();
using var timeout = new CancellationTokenSource(TimeSpan.FromMinutes(2));
try
{
    await host.StartAsync(timeout.Token);
    DurableTaskClient client = host.Services.GetRequiredService<DurableTaskClient>();
    string orderId = args.Length > 0 ? args[0] : "order-001";
    if (string.IsNullOrWhiteSpace(orderId))
    {
        throw new ArgumentException("Order ID must not be empty.");
    }

    string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
        nameof(ProcessOrder), orderId,
        new StartOrchestrationOptions { Tags = new Dictionary<string, string> { ["DTS_TaskQueue"] = "orders" } },
        timeout.Token);
    Console.WriteLine($"Started {instanceId} with DTS_TaskQueue=orders");
    OrchestrationMetadata result = await client.WaitForInstanceCompletionAsync(
        instanceId, getInputsAndOutputs: true, cancellation: timeout.Token);
    if (result.RuntimeStatus != OrchestrationRuntimeStatus.Completed)
    {
        throw new InvalidOperationException($"Instance {instanceId}: {result.RuntimeStatus}. {result.FailureDetails}");
    }

    string output = result.ReadOutputAs<string>()
        ?? throw new InvalidOperationException("Orchestration returned no output.");
    if (output != $"Recorded {orderId}; Partner accepted {orderId}; Validated {orderId}")
    {
        throw new InvalidOperationException($"Unexpected output: {output}");
    }
    Console.WriteLine(output);
}
finally
{
    using var shutdown = new CancellationTokenSource(TimeSpan.FromSeconds(30));
    await host.StopAsync(shutdown.Token);
}
