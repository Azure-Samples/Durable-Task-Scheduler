using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using DurableTaskOnAKS;

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

var builder = Host.CreateApplicationBuilder();
builder.Logging.AddConsole().SetMinimumLevel(LogLevel.Information);

string endpoint  = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHub   = Environment.GetEnvironmentVariable("TASKHUB")  ?? "default";
string? clientId = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");

string connectionString = BuildConnectionString(endpoint, taskHub, clientId);

builder.Services.AddDurableTaskWorker()
    .AddTasks(r =>
    {
        r.AddOrchestrator<DocumentProcessingOrchestration>();
        r.AddActivity<ValidateDocument>();
        r.AddActivity<ClassifyDocument>();
    })
    .UseDurableTaskScheduler(connectionString);

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

var host = builder.Build();
var log = host.Services.GetRequiredService<ILoggerFactory>().CreateLogger("Worker");

log.LogInformation("Endpoint: {Endpoint}  TaskHub: {TaskHub}", endpoint, taskHub);
log.LogInformation("Starting worker...");

await host.RunAsync();

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

static string BuildConnectionString(string endpoint, string taskHub, string? clientId)
{
    string auth = endpoint.StartsWith("http://localhost")
        ? "None"
        : !string.IsNullOrEmpty(clientId)
            ? $"ManagedIdentity;ClientID={clientId}"
            : "DefaultAzure";

    return $"Endpoint={endpoint};TaskHub={taskHub};Authentication={auth}";
}
