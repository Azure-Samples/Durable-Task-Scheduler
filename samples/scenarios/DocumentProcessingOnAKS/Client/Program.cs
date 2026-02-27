using DurableTaskOnAKS.Client.Models;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

string endpoint  = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHub   = Environment.GetEnvironmentVariable("TASKHUB")  ?? "default";
string? clientId = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");

var builder = Host.CreateApplicationBuilder();
builder.Services.AddDurableTaskClient(b => b.UseDurableTaskScheduler(
    BuildConnectionString(endpoint, taskHub, clientId)));

using var host = builder.Build();
await host.StartAsync();
var client = host.Services.GetRequiredService<DurableTaskClient>();

// ---------------------------------------------------------------------------
// Sample documents
// ---------------------------------------------------------------------------

DocumentInfo[] docs =
[
    new("doc-1", "Cloud Migration Strategy",
        "Plan to migrate on-prem workloads to Azure over 18 months."),
    new("doc-2", "Quarterly Incident Report",
        "Summary of production incidents and remediation steps for Q4."),
    new("doc-3", "ML Model Evaluation",
        "Transformer model achieved 94% accuracy on document classification."),
];

// ---------------------------------------------------------------------------
// Submit and wait for each orchestration
// ---------------------------------------------------------------------------

Console.WriteLine($"Endpoint: {endpoint} | TaskHub: {taskHub}");
Console.WriteLine($"Submitting {docs.Length} documents...\n");

foreach (var doc in docs)
{
    string id = await client.ScheduleNewOrchestrationInstanceAsync(
        "DocumentProcessingOrchestration", doc);

    Console.WriteLine($"  Scheduled [{id}] '{doc.Title}'");

    var meta = await client.WaitForInstanceCompletionAsync(id, getInputsAndOutputs: true);

    if (meta.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
        Console.WriteLine($"  -> {meta.ReadOutputAs<string>()}\n");
    else
        Console.WriteLine($"  -> {meta.RuntimeStatus}: {meta.FailureDetails?.ErrorMessage}\n");
}

Console.WriteLine("Done.");

// In non-interactive/container environments (including AKS), keep the process alive for log inspection; locally (interactive) just exit.
if (!Environment.UserInteractive || Console.IsInputRedirected)
    await Task.Delay(Timeout.InfiniteTimeSpan); // keep pod alive for log inspection

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
