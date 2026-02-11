using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

var builder = Host.CreateApplicationBuilder(args);

string connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

builder.Services.AddDurableTaskWorker()
    .AddTasks(registry => registry.AddAllGeneratedTasks())
    .UseDurableTaskScheduler(connectionString);

builder.Logging.AddSimpleConsole(options =>
{
    options.SingleLine = true;
    options.UseUtcTimestamp = true;
});

var host = builder.Build();
Console.WriteLine("Worker started. Press Ctrl+C to exit.");
await host.RunAsync();
