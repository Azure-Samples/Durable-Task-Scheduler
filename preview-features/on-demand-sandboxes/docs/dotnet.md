# On-demand Sandboxes — .NET guide

> **Status:** Private preview · [Back to overview](./README.md)

This guide walks through using On-demand Sandboxes with the **.NET** Durable Task SDK.
Make sure you've reviewed the [prerequisites](./README.md#prerequisites) first.

On-demand Sandboxes use a two-part model: a **sandbox worker profile** in your
orchestrator app that tells DTS which activities to offload, and a **worker image** that
contains those activity implementations. Your orchestrator still calls activities the
same way it always has—the decision to run one in a sandbox lives entirely in the profile
configuration.

## Step 1 — Declare a sandbox worker profile

In the app that hosts your orchestrator, define a sandbox worker profile. The profile
gives DTS the container image of your activity code, resource shape, concurrency setting,
and the activity names that should run in a sandbox.

```csharp
using Microsoft.DurableTask.Worker.AzureManaged.Sandbox;

[SandboxWorkerProfile("<worker-profile-id>")]
internal sealed class CodeSandboxWorkerProfile : ISandboxWorkerProfile
{
    public void Configure(SandboxOptions options)
    {
        options.ContainerImage = Environment.GetEnvironmentVariable("DTS_SANDBOX_IMAGE")
            ?? throw new InvalidOperationException("DTS_SANDBOX_IMAGE is required.");
        options.Cpu = "1000m";
        options.Memory = "2048Mi";
        options.MaxConcurrentActivities = 1;
        options.AddActivity(TaskNames.ExecuteCode);
    }
}
```

Then enable on-demand sandbox discovery when you configure the Durable Task worker in
the main app:

```csharp
workerBuilder.AddTasks(tasks => tasks.AddAllGeneratedTasks());
workerBuilder.UseDurableTaskScheduler(options =>
{
    options.EndpointAddress = Environment.GetEnvironmentVariable("DTS_ENDPOINT");
    options.TaskHubName = Environment.GetEnvironmentVariable("DTS_TASK_HUB");
    options.Credential = credential;
});
workerBuilder.EnableSandboxes();
```

`EnableSandboxes()` is the key line—it turns on on-demand sandbox discovery so DTS picks
up your sandbox worker profiles and routes their declared activities to managed compute.
Without it, those activities won't be offloaded.

For the meaning, accepted values, and defaults of each `SandboxOptions` setting, see the
[worker profile configuration reference](./README.md#worker-profile-configuration-reference).
In short: `ContainerImage` is the image with your activity implementations; `Cpu` /
`Memory` set the per-sandbox resource shape; `MaxConcurrentActivities` sets concurrency;
and `AddActivity` selects the activities to offload (only added activities run in
DTS-managed isolated compute; everything else stays in-process).

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
Container Registry) and reference the image in the worker profile's `ContainerImage`
option.

## Step 3 — View logs in the DTS dashboard

Once your sandbox activities are running, you can view their execution logs directly in
the Durable Task Scheduler dashboard. See
[View logs in the DTS dashboard](./README.md#view-logs-in-the-dts-dashboard) in the
overview for details.

## Next steps

- [Worker profile configuration reference](./README.md#worker-profile-configuration-reference)
- [Python guide](./python.md)
- [Back to overview](./README.md)
