// filepath: /Users/nickgreenfield1/workspace/Durable-Task-Scheduler/samples/durable-task-sdks/dotnet/HumanInteraction/Worker/Program.cs
using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

// Configure the host builder
HostApplicationBuilder builder = Host.CreateApplicationBuilder();

// Configure logging
builder.Logging.AddConsole();
builder.Logging.SetMinimumLevel(LogLevel.Information);

// Build a logger for startup configuration
using ILoggerFactory loggerFactory = LoggerFactory.Create(loggingBuilder =>
{
    loggingBuilder.AddConsole();
    loggingBuilder.SetMinimumLevel(LogLevel.Information);
});
ILogger<Program> logger = loggerFactory.CreateLogger<Program>();

string dtsConnectionString =
    builder.Configuration.GetValue<string>("DURABLE_TASK_SCHEDULER_CONNECTION_STRING")
    ?? throw new InvalidOperationException("DURABLE_TASK_SCHEDULER_CONNECTION_STRING is not set");

logger.LogInformation("Connection string: {ConnectionString}", dtsConnectionString);

// Configure services
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddAllGeneratedTasks();
    })
    .UseDurableTaskScheduler(dtsConnectionString);

// Build the host
IHost host = builder.Build();

// Start the host
await host.RunAsync();
