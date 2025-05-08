# Versioning for Azure Functions Durable Task Scheduler (preview)

Due to the nature of durable orchestration, handling different versions can be challenging for most orchestration frameworks. Maintaining a consistent event history is crucial, and any changes (like worker restarts) can lead to nondeterministic failures, causing interrupted workflows and failed deployments. To circumvent these issues, the Durable Task ecosystem provides direct version handling, allowing seamless updates without disrupting ongoing workflows.

Once versioning is enabled and set up in the client, you can perform direct version handling in:
- Versioning in the code using the Durable Task `TaskOrchestrationContext` context
- Versioning via the Durable Task worker or host 

> [!NOTE]
> Currently, direct version handling is only available for the .NET portable SDK. 

## Enable versioning in the client

In order to handle versioning with the Durable Task context or worker/host, you must first enable it in the client side, setting the value of the orchestrations being started. For example, using the portable .NET SDK, you can set the client's version as follows:

```csharp
builder.Services.AddDurableTaskClient(builder =>
{
    builder.UseDurableTaskScheduler(connectionString);
    builder.UseDefaultVersion("1.0.0");
});
```

In the previous setup example, orchestrations started by this client have version 1.0.0. While the version can be any string, the internal comparison tries to use the .NET standard. A string comparison is used if the .NET standard can't be parsed. 

## Versioning via the Durable Task context

Once you enable versioning in the client, the version is accessible in the code using the `TaskOrchestrationContext` context. The code can then make conditional decisions about the orchestration's tasks. Versioning via the context allows multiple versions to coexist, because the replay remains the same for older versions and newer (or specific) versions have the different path.

### Example

The following .NET SDK example shows how conditional workflows can avoid a nondeterministic error during execution.

```csharp
[DurableTask("HelloCities")]
class HelloCities : TaskOrchestrator<string, List<string>>
{
    private readonly string[] Cities = ["Seattle", "Amsterdam", "Kuala Lumpur", "Hyderabad", "Shanghai", "Tokyo"];

    public override async Task<List<string>> RunAsync(TaskOrchestrationContext context, string input)
    {
        List<string> results = [];
        foreach (var city in Cities)
        {
            results.Add(await context.CallSayHelloAsync($"{city} v{context.Version}"));
            if (context.CompareVersionTo("1.0.1") >= 0)
            {
                results.Add(await context.CallSayGoodbyeAsync($"{city} v{context.Version}"));
            }
        }

        Console.WriteLine("HelloCities orchestration completed.");
        return results;
    }
}

[DurableTask]
class SayHello : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string cityName)
    {
        return Task.FromResult<string>($"Hello, {cityName}!");
    }
}

[DurableTask]
class SayGoodbye : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string cityName)
    {
        return Task.FromResult<string>($"Goodbye, {cityName}!");
    }
}
```

In the previous example, we introduced a new version of the `HelloCities` orchestration that wishes to add a `Goodbye` to the orchestration. The version is:
- Accessed via the simple string `context.Version`
- Checked via the helper function in the context `CompareVersionTo`. 

If the version is greater than or equal to version 1.0.1, we can safely add in the new activity.

## Versioning via the Durable Task worker

Another way to handle versioning is via the actual worker or host. Worker versioning frees up the worker to work on orchestrations it can process by shielding it from orchestrations it can't. This streamlined process results in less overall churn than draining the old orchestrations before starting new ones.

After setting the orchestration version in the client, you provide information on the version a worker can handle in their own configuration. The worker values you need to set are:

- The version of the worker itself.
- The match strategy that the worker uses to match against the orchestration's version.
- The failure strategy that the worker should take if the version doesn't meet the matching strategy.

### Match strategies

| Name | Description |
| ---- | ----------- |
| `None` | The version isn't considered when work is being processed |
| `Strict` | The version in the orchestration and the worker must match exactly|
| `CurrentOrOlder` | The version in the orchestration must be equal to or less than the version in the worker |

### Failure strategies

| Name | Description |
| ---- | ----------- |
| `Reject` | The orchestration is rejected by the worker but remain in the work queue to be attempted again later |
| `Fail` | The orchestration is failed and removed from the work queue |

The `Reject` failure strategy is more commonly used in scenarios involving handling deployment times. For example, during a deployment, the client might be updated first, or different versions of workers might exist concurrently in the same fleet. With the `Reject` strategy, the work simply returns to the backend to be given out to a different worker.  

The `Fail` failure strategy is more commonly used in scenarios where a different version isn't expected. In this case, the orchestration shouldn't continue the "enqueue-dequeue-reenqueue" cycle. Instead, using `Fail` stops it as soon as possible so that the orchestration can be investigated in logs and dashboards.

### Example

The following example shows how to set up a worker using the required values in the .NET portable SDK.

```csharp
builder.Services.AddDurableTaskWorker(builder =>
{
    builder.AddTasks(r => r.AddAllGeneratedTasks());
    builder.UseDurableTaskScheduler(connectionString);
    builder.UseVersioning(new DurableTaskWorkerOptions.VersioningOptions
    {
        Version = "1.0.2",
        MatchStrategy = DurableTaskWorkerOptions.VersionMatchStrategy.Strict,
        FailureStrategy = DurableTaskWorkerOptions.VersionFailureStrategy.Reject,
    });
});
```

## Next steps