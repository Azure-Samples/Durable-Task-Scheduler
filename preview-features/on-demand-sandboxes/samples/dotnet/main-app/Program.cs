using Azure.Core;
using Azure.Identity;
using Demo.Codegen.MainApp;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

string endpoint = Environment.GetEnvironmentVariable("DTS_ENDPOINT")
    ?? throw new InvalidOperationException("DTS_ENDPOINT is required.");
string taskHub = Environment.GetEnvironmentVariable("DTS_TASK_HUB") ?? "default";
string aoaiEndpoint = Environment.GetEnvironmentVariable("AOAI_ENDPOINT")
    ?? throw new InvalidOperationException("AOAI_ENDPOINT is required.");
string csvPath = Environment.GetEnvironmentVariable("DEMO_CSV_PATH")
    ?? Path.Combine(AppContext.BaseDirectory, "..", "..", "..", "..", "data", "sales_q1.csv");
string question = args.Length > 0
    ? string.Join(' ', args)
    : "Which region had the highest total revenue in March 2025?";

if (!File.Exists(csvPath))
{
    Console.Error.WriteLine($"CSV file not found at: {csvPath}");
    return 1;
}

string csvData = await File.ReadAllTextAsync(csvPath);
TokenCredential credential = new DefaultAzureCredential();

HostApplicationBuilder builder = Host.CreateApplicationBuilder(args);
builder.Logging.AddSimpleConsole(options =>
{
    options.SingleLine = true;
    options.UseUtcTimestamp = true;
    options.TimestampFormat = "yyyy-MM-ddTHH:mm:ss.fffZ ";
});

builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    workerBuilder.AddTasks(tasks => tasks.AddAllGeneratedTasks());
    workerBuilder.UseWorkItemFilters();
    workerBuilder.UseDurableTaskScheduler(options =>
    {
        options.EndpointAddress = endpoint;
        options.TaskHubName = taskHub;
        options.Credential = credential;
    });
});

builder.Services.AddDurableTaskClient(clientBuilder =>
{
    clientBuilder.UseDurableTaskScheduler(options =>
    {
        options.EndpointAddress = endpoint;
        options.TaskHubName = taskHub;
        options.Credential = credential;
    });
});

// Profiles are declared in WorkerProfiles.cs via [SandboxWorkerProfile].
builder.Services.AddDurableTaskSchedulerSandboxActivitiesClient();

using IHost host = builder.Build();
await host.StartAsync();

// Declare the sandbox worker profiles with DTS so it can route ExecuteCode to a sandbox.
SandboxActivitiesClient sandboxActivitiesClient = host.Services.GetRequiredService<SandboxActivitiesClient>();
await sandboxActivitiesClient.EnableSandboxActivitiesAsync();

DurableTaskClient client = host.Services.GetRequiredService<DurableTaskClient>();

// Print demo context so the audience understands the dataset before orchestration starts.
string[] allLines = File.ReadAllLines(csvPath);
string[] headers = allLines[0].Split(',');
int rowCount = allLines.Length - 1;
string columnList = string.Join(", ", headers);
Console.WriteLine($"[demo] Dataset: {Path.GetFullPath(csvPath)}");
Console.WriteLine($"[demo] {rowCount} rows × {headers.Length} columns: [{columnList}]");
Console.WriteLine($"[demo] Preview (first 3 rows):");
int previewCount = Math.Min(3, rowCount);
for (int i = 1; i <= previewCount; i++)
{
    string[] cells = allLines[i].Split(',');
    Console.WriteLine("       " + string.Join("  ", cells));
}
Console.WriteLine($"[demo] Question: {question}");
Console.WriteLine();

string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    TaskNames.AnalyzeSalesOrchestrator,
    input: new AnalyzeSalesInput(question, csvData));

Console.WriteLine($"Started orchestration: {instanceId}");

OrchestrationMetadata? result = await client.WaitForInstanceCompletionAsync(
    instanceId,
    getInputsAndOutputs: true);

Console.WriteLine($"Status: {result?.RuntimeStatus}");
Console.WriteLine();

if (result?.FailureDetails is { } failure)
{
    Console.WriteLine($"[failure] {failure.ErrorType}: {failure.ErrorMessage}");
    if (!string.IsNullOrWhiteSpace(failure.StackTrace))
    {
        Console.WriteLine(failure.StackTrace);
    }
}
else
{
    Console.WriteLine(result?.ReadOutputAs<string>() ?? "<no output>");
}

// Schedule exactly once. Don't exit the process: this runs as an always-on Container App,
// so returning would let Container Apps restart the replica and re-run Main (scheduling a
// new orchestration every time). Instead, keep the host running so the worker stays
// connected to DTS and ready to serve sandbox work items, and block until shutdown.
Console.WriteLine();
Console.WriteLine("Orchestration complete. Worker host staying alive; press Ctrl+C to exit.");
await host.WaitForShutdownAsync();
return 0;

