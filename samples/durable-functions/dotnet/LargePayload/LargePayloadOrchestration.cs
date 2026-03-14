using System.Net;
using System.Text;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;

namespace LargePayload;

public static class LargePayloadOrchestration
{
    private const int OneMiB = 1024 * 1024;
    private const int DefaultPayloadSizeBytes = 1536 * 1024;

    [Function("StartLargePayload")]
    public static async Task<HttpResponseData> HttpStart(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", "post")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        FunctionContext executionContext)
    {
        ILogger logger = executionContext.GetLogger(nameof(HttpStart));
        int payloadSizeBytes = GetPositiveIntSetting("PAYLOAD_SIZE_BYTES", DefaultPayloadSizeBytes);
        string payload = CreatePayload(payloadSizeBytes);
        LargePayloadRequest request = new(payload, payloadSizeBytes);
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(LargePayloadRoundTrip), request);

        logger.LogInformation(
            "Started orchestration {InstanceId} with payload size {PayloadSizeBytes} bytes.",
            instanceId,
            payloadSizeBytes);

        return await client.CreateCheckStatusResponseAsync(req, instanceId);
    }

    [Function(nameof(LargePayloadRoundTrip))]
    public static async Task<LargePayloadSummary> LargePayloadRoundTrip(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        LargePayloadRequest request = context.GetInput<LargePayloadRequest>()
            ?? throw new InvalidOperationException("The orchestration input payload was not provided.");

        string echoedPayload = await context.CallActivityAsync<string>(nameof(EchoLargePayload), request.Payload)
            ?? throw new InvalidOperationException("The activity did not return a payload.");

        return new LargePayloadSummary(
            RequestedPayloadBytes: request.RequestedPayloadBytes,
            OrchestrationInputBytes: GetUtf8ByteCount(request.Payload),
            ActivityOutputBytes: GetUtf8ByteCount(echoedPayload),
            ExceededOneMiB: request.RequestedPayloadBytes > OneMiB,
            PayloadsMatch: string.Equals(request.Payload, echoedPayload, StringComparison.Ordinal));
    }

    [Function(nameof(EchoLargePayload))]
    public static string EchoLargePayload(
        [ActivityTrigger] string payload,
        FunctionContext executionContext)
    {
        ILogger logger = executionContext.GetLogger(nameof(EchoLargePayload));
        int payloadBytes = GetUtf8ByteCount(payload);

        logger.LogInformation(
            "Echoing a payload with {PayloadBytes} bytes.",
            payloadBytes);

        if (payload.StartsWith("blob:v1:", StringComparison.Ordinal))
        {
            throw new InvalidOperationException("The activity received a payload token instead of the resolved payload.");
        }

        return payload;
    }

    [Function("Hello")]
    public static async Task<HttpResponseData> Hello(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get")] HttpRequestData req)
    {
        HttpResponseData response = req.CreateResponse(HttpStatusCode.OK);
        await response.WriteStringAsync("LargePayload sample is running. POST /api/StartLargePayload to start a >1 MB orchestration.");
        return response;
    }

    private static string CreatePayload(int payloadSizeBytes) => new('L', payloadSizeBytes);

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

public sealed record LargePayloadRequest(string Payload, int RequestedPayloadBytes);

public sealed record LargePayloadSummary(
    int RequestedPayloadBytes,
    int OrchestrationInputBytes,
    int ActivityOutputBytes,
    bool ExceededOneMiB,
    bool PayloadsMatch);
