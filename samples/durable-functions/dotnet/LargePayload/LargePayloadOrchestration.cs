// Large Payload Sample - .NET Isolated Durable Functions with Durable Task Scheduler
//
// Demonstrates how to use the large payload storage feature to handle payloads
// that exceed the Durable Task Scheduler's message size limit. When enabled,
// payloads larger than the configured threshold are automatically offloaded to
// Azure Blob Storage (compressed via gzip), keeping orchestration history lean
// while supporting arbitrarily large data.
//
// This sample uses a fan-out/fan-in pattern: the orchestrator fans out to multiple
// activity functions, each of which generates a large payload (configurable size).
// The orchestrator then aggregates the results.

using System.Text.Json;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;

namespace LargePayload;

public static class LargePayloadOrchestration
{
    // Default payload size in KB (override via PAYLOAD_SIZE_KB app setting)
    private const int DefaultPayloadSizeKb = 100;

    // Default number of parallel activities (override via ACTIVITY_COUNT app setting)
    private const int DefaultActivityCount = 5;

    // -----------------------------------------------------------------------
    // HTTP Trigger – starts the orchestration
    // -----------------------------------------------------------------------
    [Function("StartLargePayload")]
    public static async Task<HttpResponseData> HttpStart(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", "post")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        FunctionContext executionContext)
    {
        ILogger logger = executionContext.GetLogger("StartLargePayload");

        // Read configuration from environment variables and pass as orchestration input
        // (environment variable access must not happen inside the orchestrator).
        int activityCount = int.TryParse(
            Environment.GetEnvironmentVariable("ACTIVITY_COUNT"), out int ac) ? ac : DefaultActivityCount;
        int payloadSizeKb = int.TryParse(
            Environment.GetEnvironmentVariable("PAYLOAD_SIZE_KB"), out int ps) ? ps : DefaultPayloadSizeKb;

        OrchestratorConfig config = new OrchestratorConfig(activityCount, payloadSizeKb);

        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(LargePayloadFanOutFanIn), config);

        logger.LogInformation("Started orchestration with ID = '{InstanceId}'.", instanceId);

        return await client.CreateCheckStatusResponseAsync(req, instanceId);
    }

    // -----------------------------------------------------------------------
    // Orchestrator – fans out to N parallel activities, each producing a large payload
    // -----------------------------------------------------------------------
    [Function(nameof(LargePayloadFanOutFanIn))]
    public static async Task<PayloadSummary> LargePayloadFanOutFanIn(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        ILogger logger = context.CreateReplaySafeLogger(nameof(LargePayloadFanOutFanIn));

        // Read config from orchestration input (set by the HTTP trigger)
        // to avoid non-deterministic environment variable access in the orchestrator.
        OrchestratorConfig config = context.GetInput<OrchestratorConfig>() ?? new OrchestratorConfig();
        int activityCount = config.ActivityCount > 0 ? config.ActivityCount : DefaultActivityCount;
        int payloadSizeKb = config.PayloadSizeKb > 0 ? config.PayloadSizeKb : DefaultPayloadSizeKb;

        logger.LogInformation(
            "Starting fan-out: {Count} activities, each generating {SizeKb} KB payloads.",
            activityCount, payloadSizeKb);

        // Fan-out: schedule N activities in parallel
        List<Task<ActivityResult>> tasks = new List<Task<ActivityResult>>();
        for (int i = 0; i < activityCount; i++)
        {
            tasks.Add(context.CallActivityAsync<ActivityResult>(
                nameof(ProcessLargeData),
                new ActivityInput(i, payloadSizeKb)));
        }

        // Fan-in: wait for all activities to complete
        ActivityResult[] results = await Task.WhenAll(tasks);

        // Aggregate results
        PayloadSummary summary = new PayloadSummary(
            ItemsProcessed: results.Length,
            TotalSizeKb: results.Sum(r => r.SizeKb),
            IndividualSizes: results.Select(r => r.SizeKb).ToArray());

        logger.LogInformation(
            "Fan-in complete: {Count} items, {TotalKb} KB total.",
            summary.ItemsProcessed, summary.TotalSizeKb);

        return summary;
    }

    // -----------------------------------------------------------------------
    // Activity – generates and returns a large payload
    // -----------------------------------------------------------------------
    [Function(nameof(ProcessLargeData))]
    public static ActivityResult ProcessLargeData(
        [ActivityTrigger] ActivityInput input,
        FunctionContext executionContext)
    {
        ILogger logger = executionContext.GetLogger(nameof(ProcessLargeData));

        logger.LogInformation(
            "Task {TaskId}: generating {SizeKb} KB payload...",
            input.TaskId, input.PayloadSizeKb);

        string payload = GenerateLargePayload(input.PayloadSizeKb);

        logger.LogInformation(
            "Task {TaskId}: payload size = {Bytes} bytes.",
            input.TaskId, payload.Length);

        return new ActivityResult(input.TaskId, input.PayloadSizeKb, payload);
    }

    // -----------------------------------------------------------------------
    // Health-check endpoint
    // -----------------------------------------------------------------------
    [Function("Hello")]
    public static HttpResponseData Hello(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get")] HttpRequestData req)
    {
        HttpResponseData response = req.CreateResponse(System.Net.HttpStatusCode.OK);
        response.WriteString("Hello from Large Payload Sample!");
        return response;
    }

    // -----------------------------------------------------------------------
    // Helper: generate a JSON payload of approximately the specified size
    // -----------------------------------------------------------------------
    private static string GenerateLargePayload(int sizeKb)
    {
        int targetBytes = sizeKb * 1024;
        // Reserve space for JSON envelope
        string filler = new('x', Math.Max(0, targetBytes - 100));
        var payload = new { size_kb = sizeKb, data = filler };
        return JsonSerializer.Serialize(payload);
    }
}

// -----------------------------------------------------------------------
// DTOs
// -----------------------------------------------------------------------
public record OrchestratorConfig(int ActivityCount = 5, int PayloadSizeKb = 100);
public record ActivityInput(int TaskId, int PayloadSizeKb);
public record ActivityResult(int TaskId, int SizeKb, string Payload);
public record PayloadSummary(int ItemsProcessed, int TotalSizeKb, int[] IndividualSizes);
