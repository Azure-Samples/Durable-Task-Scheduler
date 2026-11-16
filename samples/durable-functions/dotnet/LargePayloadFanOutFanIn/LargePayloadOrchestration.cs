using System.Net;
using System.Text;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;

namespace LargePayloadFanOutFanIn;

public static class LargePayloadOrchestration
{
    private const int OneMiB = 1024 * 1024;
    private const int DefaultPayloadSizeBytes = 1536 * 1024;
    private const int DefaultActivityCount = 3;

    [Function("StartLargePayload")]
    public static async Task<HttpResponseData> HttpStart(
        [HttpTrigger(AuthorizationLevel.Function, "get", "post")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        FunctionContext executionContext)
    {
        ILogger logger = executionContext.GetLogger(nameof(HttpStart));
        int payloadSizeBytes = GetPositiveIntSetting("PAYLOAD_SIZE_BYTES", DefaultPayloadSizeBytes);
        int activityCount = GetPositiveIntSetting("ACTIVITY_COUNT", DefaultActivityCount);
        LargePayloadFanOutRequest request = new(activityCount, payloadSizeBytes);
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(LargePayloadFanOutFanIn), request);

        logger.LogInformation(
            "Started orchestration {InstanceId} with {ActivityCount} activities at {PayloadSizeBytes} bytes each.",
            instanceId,
            activityCount,
            payloadSizeBytes);

        return await client.CreateCheckStatusResponseAsync(req, instanceId);
    }

    [Function(nameof(LargePayloadFanOutFanIn))]
    public static async Task<LargePayloadFanOutSummary> LargePayloadFanOutFanIn(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        LargePayloadFanOutRequest request = context.GetInput<LargePayloadFanOutRequest>()
            ?? throw new InvalidOperationException("The orchestration input payload was not provided.");

        List<Task<LargePayloadActivitySummary>> tasks = new(request.ActivityCount);
        for (int i = 0; i < request.ActivityCount; i++)
        {
            tasks.Add(context.CallSubOrchestratorAsync<LargePayloadActivitySummary>(
                nameof(GenerateAndValidatePayload),
                new LargePayloadActivityRequest(i + 1, request.RequestedPayloadBytes)));
        }

        LargePayloadActivitySummary[] summaries = await Task.WhenAll(tasks);
        int[] individualPayloadBytes = summaries.Select(summary => summary.PayloadBytes).ToArray();

        return new LargePayloadFanOutSummary(
            ActivityCount: summaries.Length,
            RequestedPayloadBytesPerActivity: request.RequestedPayloadBytes,
            IndividualPayloadBytes: individualPayloadBytes,
            TotalPayloadBytes: individualPayloadBytes.Sum(bytes => (long)bytes),
            AllPayloadsExceededOneMiB: individualPayloadBytes.All(bytes => bytes > OneMiB),
            AllPayloadsMatchRequestedSize: individualPayloadBytes.All(bytes => bytes == request.RequestedPayloadBytes),
            AllPayloadsMatchExpectedContent: summaries.All(summary => summary.ContentMatches));
    }

    [Function(nameof(GenerateAndValidatePayload))]
    public static async Task<LargePayloadActivitySummary> GenerateAndValidatePayload(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        LargePayloadActivityRequest request = context.GetInput<LargePayloadActivityRequest>()
            ?? throw new InvalidOperationException("The activity request was not provided.");

        // Keep large results in separate histories; the parent only receives small summaries.
        string payload = await context.CallActivityAsync<string>(nameof(GenerateLargePayload), request)
            ?? throw new InvalidOperationException("The activity did not return a payload.");
        ValidatePayload(payload, request.ActivityNumber, request.RequestedPayloadBytes);
        return new LargePayloadActivitySummary(GetUtf8ByteCount(payload), ContentMatches: true);
    }

    [Function(nameof(GenerateLargePayload))]
    public static string GenerateLargePayload(
        [ActivityTrigger] LargePayloadActivityRequest request,
        FunctionContext executionContext)
    {
        ILogger logger = executionContext.GetLogger(nameof(GenerateLargePayload));
        string payload = CreatePayload(request.ActivityNumber, request.RequestedPayloadBytes);
        int payloadBytes = GetUtf8ByteCount(payload);

        logger.LogInformation(
            "Activity {ActivityNumber} generated a payload with {PayloadBytes} bytes.",
            request.ActivityNumber,
            payloadBytes);

        return payload;
    }

    [Function("Hello")]
    public static async Task<HttpResponseData> Hello(
        [HttpTrigger(AuthorizationLevel.Function, "get")] HttpRequestData req)
    {
        HttpResponseData response = req.CreateResponse(HttpStatusCode.OK);
        await response.WriteStringAsync("LargePayloadFanOutFanIn sample is running. POST /api/StartLargePayload to start parallel >1 MB activities.");
        return response;
    }

    private static string CreatePayload(int activityNumber, int payloadSizeBytes)
    {
        // Use a deterministic, low-compressibility payload so stored blob sizes stay representative.
        return string.Create(payloadSizeBytes, 0x00C0FFEEu + (uint)activityNumber, static (span, seed) =>
        {
            const string Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
            uint state = seed;

            for (int i = 0; i < span.Length; i++)
            {
                state = (state * 1664525) + 1013904223;
                span[i] = Alphabet[(int)(state >> 26)];
            }
        });
    }

    private static void ValidatePayload(string payload, int activityNumber, int expectedSizeBytes)
    {
        if (payload.Length != expectedSizeBytes)
        {
            throw new InvalidOperationException($"Activity {activityNumber} returned an unexpected payload size.");
        }

        const string Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
        uint state = 0x00C0FFEEu + (uint)activityNumber;
        for (int i = 0; i < payload.Length; i++)
        {
            state = (state * 1664525) + 1013904223;
            if (payload[i] != Alphabet[(int)(state >> 26)])
            {
                throw new InvalidOperationException($"Activity {activityNumber} returned invalid payload content at offset {i}.");
            }
        }
    }

    private static int GetPositiveIntSetting(string key, int defaultValue)
    {
        string? rawValue = Environment.GetEnvironmentVariable(key);
        if (string.IsNullOrWhiteSpace(rawValue))
        {
            return defaultValue;
        }

        if (!int.TryParse(rawValue, out int parsedValue) || parsedValue <= 0)
        {
            throw new InvalidOperationException($"Environment variable '{key}' must be a positive integer. Value: {rawValue}");
        }

        return parsedValue;
    }

    private static int GetUtf8ByteCount(string payload) => Encoding.UTF8.GetByteCount(payload);
}

public sealed record LargePayloadFanOutRequest(int ActivityCount, int RequestedPayloadBytes);

public sealed record LargePayloadActivityRequest(int ActivityNumber, int RequestedPayloadBytes);

public sealed record LargePayloadActivitySummary(int PayloadBytes, bool ContentMatches);

public sealed record LargePayloadFanOutSummary(
    int ActivityCount,
    int RequestedPayloadBytesPerActivity,
    int[] IndividualPayloadBytes,
    long TotalPayloadBytes,
    bool AllPayloadsExceededOneMiB,
    bool AllPayloadsMatchRequestedSize,
    bool AllPayloadsMatchExpectedContent);
