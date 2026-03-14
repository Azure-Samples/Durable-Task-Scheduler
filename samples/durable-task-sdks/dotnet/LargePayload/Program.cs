// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

using Azure.Core;
using Azure.Identity;
using Azure.Storage.Blobs;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

const int OneMiB = 1024 * 1024;
const int DefaultPayloadSizeBytes = 1536 * 1024;
const int DefaultExternalizeThresholdBytes = 900_000;
const string OrchestrationName = "LargePayloadRoundTrip";
const string ActivityName = "EchoLargePayload";

HostApplicationBuilder builder = Host.CreateApplicationBuilder(args);

string schedulerConnectionString = builder.Configuration.GetValue<string>("DURABLE_TASK_SCHEDULER_CONNECTION_STRING")
    ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

int payloadSizeBytes = GetPositiveInt(builder.Configuration, "PAYLOAD_SIZE_BYTES", DefaultPayloadSizeBytes);
int externalizeThresholdBytes = GetPositiveInt(builder.Configuration, "EXTERNALIZE_THRESHOLD_BYTES", DefaultExternalizeThresholdBytes);

if (externalizeThresholdBytes > OneMiB)
{
    throw new InvalidOperationException($"EXTERNALIZE_THRESHOLD_BYTES must be 1 MiB or smaller. Value: {externalizeThresholdBytes}");
}

PayloadStorageSettings payloadStorageSettings = GetPayloadStorageSettings(builder.Configuration);

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

using IHost host = builder.Build();
await host.StartAsync();

await using DurableTaskClient client = host.Services.GetRequiredService<DurableTaskClient>();
BlobContainerClient payloadContainerClient = CreatePayloadContainerClient(payloadStorageSettings);
int payloadBlobCountBeforeRun = await GetBlobCountAsync(payloadContainerClient);

string largePayload = CreatePayload(payloadSizeBytes);
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(OrchestrationName, largePayload);

Console.WriteLine("Large payload round-trip sample");
Console.WriteLine($"Scheduler connection: {schedulerConnectionString}");
Console.WriteLine($"Payload bytes: {GetUtf8ByteCount(largePayload):N0}");
Console.WriteLine($"Payload exceeds 1 MiB: {GetUtf8ByteCount(largePayload) > OneMiB}");
Console.WriteLine($"Externalize threshold bytes: {externalizeThresholdBytes:N0}");
Console.WriteLine($"Instance ID: {instanceId}");

using CancellationTokenSource cancellationTokenSource = new(TimeSpan.FromMinutes(2));
OrchestrationMetadata completed = await client.WaitForInstanceCompletionAsync(
    instanceId,
    getInputsAndOutputs: true,
    cancellationTokenSource.Token);

string echoedPayload = completed.ReadOutputAs<string>() ?? string.Empty;
int payloadBlobCountAfterRun = await GetBlobCountAsync(payloadContainerClient);
int newPayloadBlobCount = payloadBlobCountAfterRun - payloadBlobCountBeforeRun;

Console.WriteLine($"Runtime status: {completed.RuntimeStatus}");
Console.WriteLine($"Payload blobs added during run: {newPayloadBlobCount}");
Console.WriteLine($"Payload offload observed: {newPayloadBlobCount > 0}");
Console.WriteLine($"Output bytes: {GetUtf8ByteCount(echoedPayload):N0}");
Console.WriteLine($"Round-trip payload matched: {string.Equals(largePayload, echoedPayload, StringComparison.Ordinal)}");
Console.WriteLine("List blobs in durabletask-payloads to verify payload offload.");

await host.StopAsync();

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

static async Task<int> GetBlobCountAsync(BlobContainerClient blobContainerClient)
{
    if (!await blobContainerClient.ExistsAsync())
    {
        return 0;
    }

    int count = 0;
    await foreach (var _ in blobContainerClient.GetBlobsAsync())
    {
        count++;
    }

    return count;
}

static string CreatePayload(int payloadSizeBytes) => new('B', payloadSizeBytes);

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
