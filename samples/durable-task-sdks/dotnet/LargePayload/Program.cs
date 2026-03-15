// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

using Azure.Core;
using Azure.Identity;
using Azure.Storage.Blobs;
using Azure.Storage.Blobs.Models;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;

const int OneMiB = 1024 * 1024;
const int DefaultPayloadSizeBytes = 1536 * 1024;
const int DefaultExternalizeThresholdBytes = 900_000;
const int DefaultWaitTimeoutSeconds = 120;
const string OrchestrationName = "LargePayloadRoundTrip";
const string ActivityName = "EchoLargePayload";

WebApplicationBuilder builder = WebApplication.CreateBuilder(args);

string schedulerConnectionString = builder.Configuration.GetValue<string>("DURABLE_TASK_SCHEDULER_CONNECTION_STRING")
    ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

int defaultPayloadSizeBytes = GetPositiveInt(builder.Configuration, "PAYLOAD_SIZE_BYTES", DefaultPayloadSizeBytes);
int externalizeThresholdBytes = GetPositiveInt(builder.Configuration, "EXTERNALIZE_THRESHOLD_BYTES", DefaultExternalizeThresholdBytes);

if (externalizeThresholdBytes > OneMiB)
{
    throw new InvalidOperationException($"EXTERNALIZE_THRESHOLD_BYTES must be 1 MiB or smaller. Value: {externalizeThresholdBytes}");
}

PayloadStorageSettings payloadStorageSettings = GetPayloadStorageSettings(builder.Configuration);
LargePayloadRuntimeSettings runtimeSettings = new(defaultPayloadSizeBytes, externalizeThresholdBytes, payloadStorageSettings.ContainerName);

builder.Services.AddSingleton(runtimeSettings);
builder.Services.AddSingleton(payloadStorageSettings);
builder.Services.AddSingleton(CreatePayloadContainerClient(payloadStorageSettings));

builder.Services.AddLogging(logging =>
{
    logging.AddSimpleConsole(options =>
    {
        options.SingleLine = true;
        options.UseUtcTimestamp = true;
        options.TimestampFormat = "yyyy-MM-ddTHH:mm:ss.fffZ ";
    });
});

builder.Services.AddExternalizedPayloadStore(options =>
{
    options.ExternalizeThresholdBytes = externalizeThresholdBytes;
    options.ContainerName = payloadStorageSettings.ContainerName;

    if (!string.IsNullOrWhiteSpace(payloadStorageSettings.ConnectionString))
    {
        options.ConnectionString = payloadStorageSettings.ConnectionString;
    }
    else
    {
        options.AccountUri = payloadStorageSettings.AccountUri;
        options.Credential = payloadStorageSettings.Credential;
    }
});

builder.Services.AddDurableTaskClient(client =>
{
    client.UseDurableTaskScheduler(schedulerConnectionString);
    client.UseExternalizedPayloads();
});

builder.Services.AddDurableTaskWorker(worker =>
{
    worker.UseDurableTaskScheduler(schedulerConnectionString);
    worker.AddTasks(tasks =>
    {
        tasks.AddOrchestratorFunc<string, string>(OrchestrationName, async (context, input) =>
        {
            string echoedPayload = await context.CallActivityAsync<string>(ActivityName, input)
                ?? throw new InvalidOperationException("The activity did not return a payload.");
            return echoedPayload;
        });

        tasks.AddActivityFunc<string, string>(ActivityName, (_, payload) =>
        {
            if (payload.StartsWith("blob:v1:", StringComparison.Ordinal))
            {
                throw new InvalidOperationException("The activity received a payload token instead of the resolved payload.");
            }

            return payload;
        });
    });

    worker.UseExternalizedPayloads();
});

WebApplication app = builder.Build();

app.MapGet("/", (LargePayloadRuntimeSettings settings) =>
    Results.Ok(new SampleInfoResponse(
        Name: "Large Payload round-trip sample",
        Description: "Trigger a >1 MiB orchestration payload and verify blob offload through the Durable Task SDK.",
        DefaultPayloadSizeBytes: settings.DefaultPayloadSizeBytes,
        ExternalizeThresholdBytes: settings.ExternalizeThresholdBytes,
        PayloadContainerName: settings.PayloadContainerName,
        RunEndpoint: "/api/largepayload/run",
        StatusEndpointTemplate: "/api/largepayload/instances/{instanceId}",
        HealthEndpoint: "/healthz")));

app.MapGet("/healthz", () => Results.Ok(new { status = "Healthy" }));

app.MapPost("/api/largepayload/run", async (
    RunLargePayloadRequest? request,
    DurableTaskClient client,
    BlobContainerClient payloadContainerClient,
    LargePayloadRuntimeSettings settings,
    CancellationToken cancellationToken) =>
{
    int requestedPayloadSizeBytes = request?.PayloadSizeBytes ?? settings.DefaultPayloadSizeBytes;
    if (requestedPayloadSizeBytes <= 0)
    {
        return Results.ValidationProblem(new Dictionary<string, string[]>
        {
            [nameof(RunLargePayloadRequest.PayloadSizeBytes)] = ["PayloadSizeBytes must be a positive integer."],
        });
    }

    int waitTimeoutSeconds = request?.WaitTimeoutSeconds ?? DefaultWaitTimeoutSeconds;
    if (waitTimeoutSeconds <= 0)
    {
        return Results.ValidationProblem(new Dictionary<string, string[]>
        {
            [nameof(RunLargePayloadRequest.WaitTimeoutSeconds)] = ["WaitTimeoutSeconds must be a positive integer."],
        });
    }

    string largePayload = CreatePayload(requestedPayloadSizeBytes);
    PayloadContainerStats payloadStatsBeforeRun = await GetPayloadContainerStatsAsync(payloadContainerClient, cancellationToken);
    string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(OrchestrationName, largePayload);
    string statusQueryGetUri = $"/api/largepayload/instances/{instanceId}";

    app.Logger.LogInformation(
        "Started large payload verification. InstanceId={InstanceId}, PayloadBytes={PayloadBytes}, ThresholdBytes={ThresholdBytes}",
        instanceId,
        requestedPayloadSizeBytes,
        settings.ExternalizeThresholdBytes);

    using CancellationTokenSource waitForCompletionSource = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
    waitForCompletionSource.CancelAfter(TimeSpan.FromSeconds(waitTimeoutSeconds));

    try
    {
        OrchestrationMetadata completed = await client.WaitForInstanceCompletionAsync(
            instanceId,
            getInputsAndOutputs: true,
            waitForCompletionSource.Token);

        string echoedPayload = completed.ReadOutputAs<string>() ?? string.Empty;
        PayloadContainerStats payloadStatsAfterRun = await GetPayloadContainerStatsAsync(payloadContainerClient, cancellationToken);
        int newPayloadBlobCount = payloadStatsAfterRun.Count - payloadStatsBeforeRun.Count;
        long newStoredPayloadBytes = payloadStatsAfterRun.TotalStoredBytes - payloadStatsBeforeRun.TotalStoredBytes;
        bool payloadsMatch = string.Equals(largePayload, echoedPayload, StringComparison.Ordinal);

        app.Logger.LogInformation(
            "Completed large payload verification. InstanceId={InstanceId}, RuntimeStatus={RuntimeStatus}, OffloadObserved={OffloadObserved}, StoredPayloadBytesAdded={StoredPayloadBytesAdded}, PayloadMatched={PayloadMatched}",
            instanceId,
            completed.RuntimeStatus,
            newPayloadBlobCount > 0,
            newStoredPayloadBytes,
            payloadsMatch);

        return Results.Ok(new RunLargePayloadResponse(
            InstanceId: instanceId,
            RuntimeStatus: completed.RuntimeStatus.ToString(),
            RequestedPayloadBytes: GetUtf8ByteCount(largePayload),
            PayloadExceedsOneMiB: GetUtf8ByteCount(largePayload) > OneMiB,
            ExternalizeThresholdBytes: settings.ExternalizeThresholdBytes,
            PayloadContainerName: settings.PayloadContainerName,
            PayloadBlobCountBeforeRun: payloadStatsBeforeRun.Count,
            PayloadBlobCountAfterRun: payloadStatsAfterRun.Count,
            PayloadBlobsAddedDuringRun: newPayloadBlobCount,
            PayloadStoredBytesBeforeRun: payloadStatsBeforeRun.TotalStoredBytes,
            PayloadStoredBytesAfterRun: payloadStatsAfterRun.TotalStoredBytes,
            PayloadStoredBytesAddedDuringRun: newStoredPayloadBytes,
            PayloadOffloadObserved: newPayloadBlobCount > 0,
            OutputBytes: GetUtf8ByteCount(echoedPayload),
            RoundTripPayloadMatched: payloadsMatch,
            StatusQueryGetUri: statusQueryGetUri));
    }
    catch (OperationCanceledException) when (!cancellationToken.IsCancellationRequested)
    {
        app.Logger.LogWarning(
            "Large payload verification timed out while waiting for completion. InstanceId={InstanceId}, WaitTimeoutSeconds={WaitTimeoutSeconds}",
            instanceId,
            waitTimeoutSeconds);

        return Results.Accepted(
            uri: statusQueryGetUri,
            value: new AcceptedRunResponse(
                InstanceId: instanceId,
                RuntimeStatus: "Running",
                RequestedPayloadBytes: GetUtf8ByteCount(largePayload),
                PayloadExceedsOneMiB: GetUtf8ByteCount(largePayload) > OneMiB,
                ExternalizeThresholdBytes: settings.ExternalizeThresholdBytes,
                PayloadContainerName: settings.PayloadContainerName,
                WaitTimeoutSeconds: waitTimeoutSeconds,
                StatusQueryGetUri: statusQueryGetUri));
    }
});

app.MapGet("/api/largepayload/instances/{instanceId}", async (
    string instanceId,
    DurableTaskClient client,
    BlobContainerClient payloadContainerClient,
    LargePayloadRuntimeSettings settings,
    CancellationToken cancellationToken) =>
{
    OrchestrationMetadata? metadata = await client.GetInstanceAsync(instanceId, getInputsAndOutputs: true, cancellationToken);
    if (metadata is null)
    {
        return Results.NotFound();
    }

    string? output = metadata.RuntimeStatus == OrchestrationRuntimeStatus.Completed
        ? metadata.ReadOutputAs<string>()
        : null;

    int? outputBytes = output is not null ? GetUtf8ByteCount(output) : null;
    PayloadContainerStats payloadStats = await GetPayloadContainerStatsAsync(payloadContainerClient, cancellationToken);

    return Results.Ok(new LargePayloadStatusResponse(
        InstanceId: metadata.InstanceId,
        RuntimeStatus: metadata.RuntimeStatus.ToString(),
        CreatedAt: metadata.CreatedAt,
        PayloadContainerName: settings.PayloadContainerName,
        CurrentPayloadBlobCount: payloadStats.Count,
        CurrentPayloadStoredBytes: payloadStats.TotalStoredBytes,
        OutputBytes: outputBytes,
        FailureMessage: metadata.FailureDetails?.ErrorMessage));
});

app.Run();

static TokenCredential CreateCredential(IConfiguration configuration)
{
    string? managedIdentityClientId = configuration.GetValue<string>("PAYLOAD_STORAGE_MANAGED_IDENTITY_CLIENT_ID")
        ?? configuration.GetValue<string>("AZURE_CLIENT_ID");

    if (string.IsNullOrWhiteSpace(managedIdentityClientId))
    {
        return new DefaultAzureCredential();
    }

    return new DefaultAzureCredential(new DefaultAzureCredentialOptions
    {
        ManagedIdentityClientId = managedIdentityClientId,
    });
}

static BlobContainerClient CreatePayloadContainerClient(PayloadStorageSettings payloadStorageSettings)
{
    if (!string.IsNullOrWhiteSpace(payloadStorageSettings.ConnectionString))
    {
        return new BlobContainerClient(payloadStorageSettings.ConnectionString, payloadStorageSettings.ContainerName);
    }

    BlobServiceClient blobServiceClient = new(payloadStorageSettings.AccountUri, payloadStorageSettings.Credential);
    return blobServiceClient.GetBlobContainerClient(payloadStorageSettings.ContainerName);
}

static async Task<PayloadContainerStats> GetPayloadContainerStatsAsync(BlobContainerClient blobContainerClient, CancellationToken cancellationToken)
{
    if (!await blobContainerClient.ExistsAsync(cancellationToken))
    {
        return new PayloadContainerStats(0, 0);
    }

    int count = 0;
    long totalStoredBytes = 0;
    await foreach (BlobItem blob in blobContainerClient.GetBlobsAsync(cancellationToken: cancellationToken))
    {
        count++;
        totalStoredBytes += blob.Properties.ContentLength ?? 0;
    }

    return new PayloadContainerStats(count, totalStoredBytes);
}

static string CreatePayload(int payloadSizeBytes)
{
    // Use a deterministic, low-compressibility payload so the stored blob sizes stay representative.
    return string.Create(payloadSizeBytes, 0x00C0FFEEu, static (span, seed) =>
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

static int GetPositiveInt(IConfiguration configuration, string key, int defaultValue)
{
    string? rawValue = configuration.GetValue<string>(key);
    if (string.IsNullOrWhiteSpace(rawValue))
    {
        return defaultValue;
    }

    if (!int.TryParse(rawValue, out int parsedValue) || parsedValue <= 0)
    {
        throw new InvalidOperationException($"Configuration value '{key}' must be a positive integer. Value: {rawValue}");
    }

    return parsedValue;
}

static int GetUtf8ByteCount(string payload) => System.Text.Encoding.UTF8.GetByteCount(payload);

static PayloadStorageSettings GetPayloadStorageSettings(IConfiguration configuration)
{
    string containerName = configuration.GetValue<string>("PAYLOAD_CONTAINER_NAME")
        ?? configuration.GetValue<string>("DURABLETASK_PAYLOAD_CONTAINER")
        ?? "durabletask-payloads";

    string? storageConnectionString = configuration.GetValue<string>("PAYLOAD_STORAGE_CONNECTION_STRING")
        ?? configuration.GetValue<string>("DURABLETASK_STORAGE");

    if (!string.IsNullOrWhiteSpace(storageConnectionString))
    {
        return new PayloadStorageSettings(containerName, storageConnectionString, null, null);
    }

    string? storageAccountUri = configuration.GetValue<string>("PAYLOAD_STORAGE_ACCOUNT_URI");
    if (!string.IsNullOrWhiteSpace(storageAccountUri))
    {
        return new PayloadStorageSettings(
            containerName,
            null,
            new Uri(storageAccountUri),
            CreateCredential(configuration));
    }

    return new PayloadStorageSettings(containerName, "UseDevelopmentStorage=true", null, null);
}

sealed record PayloadStorageSettings(
    string ContainerName,
    string? ConnectionString,
    Uri? AccountUri,
    TokenCredential? Credential);

sealed record LargePayloadRuntimeSettings(
    int DefaultPayloadSizeBytes,
    int ExternalizeThresholdBytes,
    string PayloadContainerName);

sealed record PayloadContainerStats(
    int Count,
    long TotalStoredBytes);

sealed record SampleInfoResponse(
    string Name,
    string Description,
    int DefaultPayloadSizeBytes,
    int ExternalizeThresholdBytes,
    string PayloadContainerName,
    string RunEndpoint,
    string StatusEndpointTemplate,
    string HealthEndpoint);

sealed record RunLargePayloadRequest(
    int? PayloadSizeBytes,
    int? WaitTimeoutSeconds);

sealed record RunLargePayloadResponse(
    string InstanceId,
    string RuntimeStatus,
    int RequestedPayloadBytes,
    bool PayloadExceedsOneMiB,
    int ExternalizeThresholdBytes,
    string PayloadContainerName,
    int PayloadBlobCountBeforeRun,
    int PayloadBlobCountAfterRun,
    int PayloadBlobsAddedDuringRun,
    long PayloadStoredBytesBeforeRun,
    long PayloadStoredBytesAfterRun,
    long PayloadStoredBytesAddedDuringRun,
    bool PayloadOffloadObserved,
    int OutputBytes,
    bool RoundTripPayloadMatched,
    string StatusQueryGetUri);

sealed record AcceptedRunResponse(
    string InstanceId,
    string RuntimeStatus,
    int RequestedPayloadBytes,
    bool PayloadExceedsOneMiB,
    int ExternalizeThresholdBytes,
    string PayloadContainerName,
    int WaitTimeoutSeconds,
    string StatusQueryGetUri);

sealed record LargePayloadStatusResponse(
    string InstanceId,
    string RuntimeStatus,
    DateTimeOffset CreatedAt,
    string PayloadContainerName,
    int CurrentPayloadBlobCount,
    long CurrentPayloadStoredBytes,
    int? OutputBytes,
    string? FailureMessage);
