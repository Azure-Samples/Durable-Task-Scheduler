using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

var builder = Host.CreateApplicationBuilder(args);

builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddAllGeneratedTasks();
    })
    .UseDurableTaskScheduler("Endpoint=http://localhost:8080;TaskHub=default;Authentication=None");

builder.Logging.AddSimpleConsole(options =>
{
    options.SingleLine = true;
    options.UseUtcTimestamp = true;
});

var host = builder.Build();
Console.WriteLine("Worker started. Press Ctrl+C to exit.");
await host.RunAsync();
