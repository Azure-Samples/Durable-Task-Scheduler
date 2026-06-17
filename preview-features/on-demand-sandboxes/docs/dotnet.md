# On-demand Sandboxes — .NET guide

> **Status:** Private preview · [Back to overview](./README.md)

This guide walks through using On-demand Sandboxes with the **.NET** Durable Task SDK.
Make sure you've reviewed the [prerequisites](./README.md#prerequisites) first.

On-demand Sandboxes use a two-part model: a **sandbox worker profile** in your
orchestrator app that tells DTS which activities to offload, and a **worker image** that
contains those activity implementations. Your orchestrator still calls activities the
same way it always has—the decision to run one in a sandbox lives entirely in the profile
configuration.

## Install the preview packages

The on-demand sandbox APIs ship in two opt-in preview packages that layer on top of the
Azure-managed client and worker packages:

- `Microsoft.DurableTask.Client.AzureManaged.Sandboxes` — declarer-app side
  (`[SandboxWorkerProfile]`, `SandboxWorkerProfileOptions`, `SandboxActivitiesClient`).
- `Microsoft.DurableTask.Worker.AzureManaged.Sandboxes` — sandbox-worker side
  (`UseSandboxWorker()`).

Add the client and worker packages to your orchestrator app, and the worker package to
the sandbox worker image project:

```bash
# Orchestrator / declarer app
dotnet add package Microsoft.DurableTask.Client.AzureManaged.Sandboxes --version 1.25.0-preview.2
dotnet add package Microsoft.DurableTask.Worker.AzureManaged.Sandboxes --version 1.25.0-preview.2

# Sandbox worker image project
dotnet add package Microsoft.DurableTask.Worker.AzureManaged.Sandboxes --version 1.25.0-preview.2
```

## Step 1 — Declare a sandbox worker profile

In the app that hosts your orchestrator, define a sandbox worker profile. The profile
gives DTS the container image of your activity code, the managed identities DTS uses to
pull the image and start the sandbox, the resource shape, concurrency setting, and the
activity names that should run in a sandbox.

```csharp
using Microsoft.DurableTask.Client.AzureManaged;

[SandboxWorkerProfile("<worker-profile-id>")]
internal sealed class CodeSandboxWorkerProfile : ISandboxWorkerProfile
{
    public void Configure(SandboxWorkerProfileOptions options)
    {
        options.Image.ImageRef = Environment.GetEnvironmentVariable("DTS_SANDBOX_CONTAINER_IMAGE")
            ?? throw new InvalidOperationException("DTS_SANDBOX_CONTAINER_IMAGE is required.");
        options.Image.ManagedIdentityClientId = Environment.GetEnvironmentVariable("DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID")
            ?? throw new InvalidOperationException("DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID is required.");
        options.SchedulerManagedIdentityClientId = Environment.GetEnvironmentVariable("DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID")
            ?? throw new InvalidOperationException("DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID is required.");
        options.Cpu = "1000m";
        options.Memory = "2048Mi";
        options.MaxConcurrentActivities = 1;
        options.AddActivity(TaskNames.ExecuteCode, version: "");
    }
}
```

> [!IMPORTANT]
> The managed identities referenced by `options.Image.ManagedIdentityClientId` and
> `options.SchedulerManagedIdentityClientId` must both be attached to the scheduler. The
> image-pull identity must have the **AcrPull** role on your container registry, and the
> worker/scheduler identity must have whatever roles your activity code needs on the
> downstream services it calls. You can use the same identity for both or split them. See
> [Configure the scheduler identity for image pull](./README.md#configure-the-scheduler-identity-for-image-pull).

Then, in the main app, enable work-item filters, register the sandbox activities client,
and declare the profiles with DTS:

```csharp
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

// Profiles are declared via [SandboxWorkerProfile]. This registers the client that
// publishes them to DTS.
builder.Services.AddDurableTaskSchedulerSandboxActivitiesClient();
```

`UseWorkItemFilters()` is required: without it, DTS can dispatch a sandbox activity to
your in-process worker—which doesn't implement it—and the orchestration gets stuck
retrying the wrong worker.

Once the host is running, publish the declared profiles to DTS so it can route their
activities to a sandbox:

```csharp
SandboxActivitiesClient sandboxActivitiesClient =
    host.Services.GetRequiredService<SandboxActivitiesClient>();
await sandboxActivitiesClient.EnableSandboxActivitiesAsync();
```

`EnableSandboxActivitiesAsync()` is the key call—it registers your sandbox worker profiles
with DTS so it picks them up and routes their declared activities to managed compute.
Without it, those activities won't be offloaded.

For the meaning, accepted values, and defaults of each `SandboxWorkerProfileOptions`
setting, see the
[worker profile configuration reference](./README.md#worker-profile-configuration-reference).
In short: `Image.ImageRef` is the image with your activity implementations;
`Image.ManagedIdentityClientId` is the managed identity DTS uses to **pull the worker
image** from your registry (needs **AcrPull**), while `SchedulerManagedIdentityClientId`
is the managed identity the **sandbox worker uses to connect back to DTS** and that the
activity code runs as when it calls other services; `Cpu` / `Memory` set the per-sandbox
resource shape; `MaxConcurrentActivities` sets concurrency; and `AddActivity` selects the
activities to offload (only added activities run in DTS-managed isolated compute;
everything else stays in-process).

The orchestrator call site doesn't change:

```csharp
ExecuteCodeOutput execution = await context.CallActivityAsync<ExecuteCodeOutput>(
    TaskNames.ExecuteCode,
    new ExecuteCodeInput(pythonCode, input.CsvData));
```

Because `ExecuteCode` is not registered in the main app's in-process activity list, DTS
uses the profile to route the work to the sandbox image when the orchestrator calls it.

## Step 2 — Build the worker image

The worker image is a container you own. In most apps, this worker lives in a separate
project from the orchestrator host so it can have its own entry point, dependencies, and
container image. It registers the activity implementations it can run and opts in to
managed execution with `UseSandboxWorker()`:

```csharp
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    workerBuilder.AddTasks(tasks =>
    {
        tasks.AddActivity<ExecuteCodeActivity>();
    });

    workerBuilder.UseSandboxWorker();
});
```

`UseSandboxWorker()` is the key line—it signals that this worker runs in DTS-managed
compute. The sandbox worker does **not** need to configure the DTS endpoint, task hub,
profile id, or credentials; DTS injects the runtime settings when it starts the
container.

The activity implementations themselves are standard Durable Task activities. There's
nothing special about the activity code—it can call a runtime with different
dependencies (for example, Python and pandas) while running in an isolated container
instead of in your main app's process.

Package the image like any containerized service, including whatever runtimes and native
tools the activity needs. Push it to your container registry (for example, Azure
Container Registry) and reference the image in the worker profile's `Image.ImageRef`
option. The image-pull identity you set in `Image.ManagedIdentityClientId` must have the
**AcrPull** role on that registry.

## Step 3 — View logs in the DTS dashboard

Once your sandbox activities are running, you can view their execution logs directly in
the Durable Task Scheduler dashboard. See
[View logs in the DTS dashboard](./README.md#view-logs-in-the-dts-dashboard) in the
overview for details.

## Next steps

- [Worker profile configuration reference](./README.md#worker-profile-configuration-reference)
- [Configure the scheduler identity for image pull](./README.md#configure-the-scheduler-identity-for-image-pull)
- [End-to-end .NET sample](../samples/dotnet)
- [Python guide](./python.md)
- [Back to overview](./README.md)
