using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

const string WorkerName = "Validator Worker";

HostApplicationBuilder builder = Host.CreateApplicationBuilder();

builder.Logging.AddSimpleConsole(options =>
{
    options.SingleLine = true;
    options.TimestampFormat = "HH:mm:ss ";
});
builder.Logging.SetMinimumLevel(LogLevel.Information);

using ILoggerFactory startupLoggerFactory = LoggerFactory.Create(lb =>
{
    lb.AddSimpleConsole(o => { o.SingleLine = true; o.TimestampFormat = "HH:mm:ss "; });
    lb.SetMinimumLevel(LogLevel.Information);
});
ILogger logger = startupLoggerFactory.CreateLogger(WorkerName);

string endpoint = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHubName = Environment.GetEnvironmentVariable("TASKHUB") ?? "default";
string? managedIdentityClientId = Environment.GetEnvironmentVariable("AZURE_MANAGED_IDENTITY_CLIENT_ID");
bool isLocalEmulator = endpoint == "http://localhost:8080";

string connectionString = isLocalEmulator
    ? $"Endpoint={endpoint};TaskHub={taskHubName};Authentication=None"
    : !string.IsNullOrEmpty(managedIdentityClientId)
        ? $"Endpoint={endpoint};TaskHub={taskHubName};Authentication=ManagedIdentity;ClientID={managedIdentityClientId}"
        : $"Endpoint={endpoint};TaskHub={taskHubName};Authentication=DefaultAzure";

logger.LogInformation("[{Worker}] Connection: {ConnectionString}", WorkerName, connectionString);
logger.LogInformation("[{Worker}] This worker registers ONLY the ValidateOrder activity.", WorkerName);

builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        // Only ValidateOrder is registered here.
        // Work item filters are auto-generated from the registry, so this worker
        // will ONLY receive ValidateOrder activity work items.
        registry.AddAllGeneratedTasks();
    })
    .UseDurableTaskScheduler(connectionString);

IHost host = builder.Build();
logger = host.Services.GetRequiredService<ILogger<Program>>();

logger.LogInformation("[{Worker}] Starting... waiting for ValidateOrder activity work items only.", WorkerName);

await host.RunAsync();
