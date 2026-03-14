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
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", "post")] HttpRequestData req,
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

        List<Task<string>> tasks = new(request.ActivityCount);
        for (int i = 0; i < request.ActivityCount; i++)
        {
            tasks.Add(context.CallActivityAsync<string>(
                nameof(GenerateLargePayload),
                new LargePayloadActivityRequest(i + 1, request.RequestedPayloadBytes)));
        }

        string[] payloads = await Task.WhenAll(tasks);
        if (payloads.Any(payload => payload.StartsWith("blob:v1:", StringComparison.Ordinal)))
        {
            throw new InvalidOperationException("The orchestrator received a payload token instead of the resolved payload.");
        }

        int[] individualPayloadBytes = payloads.Select(GetUtf8ByteCount).ToArray();

        return new LargePayloadFanOutSummary(
            ActivityCount: payloads.Length,
            RequestedPayloadBytesPerActivity: request.RequestedPayloadBytes,
            IndividualPayloadBytes: individualPayloadBytes,
            TotalPayloadBytes: individualPayloadBytes.Sum(),
            AllPayloadsExceededOneMiB: individualPayloadBytes.All(bytes => bytes > OneMiB),
            AllPayloadsMatchRequestedSize: individualPayloadBytes.All(bytes => bytes == request.RequestedPayloadBytes));
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
        [HttpTrigger(AuthorizationLevel.Anonymous, "get")] HttpRequestData req)
    {
        HttpResponseData response = req.CreateResponse(HttpStatusCode.OK);
        await response.WriteStringAsync("LargePayloadFanOutFanIn sample is running. POST /api/StartLargePayload to start parallel >1 MB activities.");
        return response;
    }

    private static string CreatePayload(int activityNumber, int payloadSizeBytes)
    {
        char payloadCharacter = (char)('A' + ((activityNumber - 1) % 26));
        return new string(payloadCharacter, payloadSizeBytes);
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

public sealed record LargePayloadFanOutSummary(
    int ActivityCount,
    int RequestedPayloadBytesPerActivity,
    int[] IndividualPayloadBytes,
    int TotalPayloadBytes,
    bool AllPayloadsExceededOneMiB,
    bool AllPayloadsMatchRequestedSize);
