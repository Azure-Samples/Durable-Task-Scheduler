# On-demand Sandboxes: .NET guide

> **Status:** Private preview · [Back to overview](./README.md)

This guide walks through using On-demand Sandboxes with the **.NET** Durable Task SDK.
Make sure you've read the [overview](./README.md) first.

On-demand Sandboxes use a two-part model: a **sandbox worker profile** in your
orchestrator app that tells DTS which activities to offload, and a **worker image** that
contains those activity implementations. Your orchestrator still calls activities the
same way it always has. The decision to run one in a sandbox lives entirely in the profile
configuration.

## Install the preview packages

The on-demand sandbox APIs ship in two opt-in preview packages that layer on top of the
Azure-managed client and worker packages:

- `Microsoft.DurableTask.Client.AzureManaged.Sandboxes`, the declarer-app side
  (`[SandboxWorkerProfile]`, `SandboxWorkerProfileOptions`, `SandboxActivitiesClient`).
- `Microsoft.DurableTask.Worker.AzureManaged.Sandboxes`, the sandbox-worker side
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

## Step 1: Declare a sandbox worker profile

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
> [Configure the scheduler identity for image pull](#configure-the-scheduler-identity-for-image-pull).

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
your in-process worker, which doesn't implement it, and the orchestration gets stuck
retrying the wrong worker.

Once the host is running, publish the declared profiles to DTS so it can route their
activities to a sandbox:

```csharp
SandboxActivitiesClient sandboxActivitiesClient =
    host.Services.GetRequiredService<SandboxActivitiesClient>();
await sandboxActivitiesClient.EnableSandboxActivitiesAsync();
```

`EnableSandboxActivitiesAsync()` is the key call. It registers your sandbox worker profiles
with DTS so it picks them up and routes their declared activities to managed compute.
Without it, those activities won't be offloaded.

The orchestrator call site doesn't change:

```csharp
ExecuteCodeOutput execution = await context.CallActivityAsync<ExecuteCodeOutput>(
    TaskNames.ExecuteCode,
    new ExecuteCodeInput(pythonCode, input.CsvData));
```

Because `ExecuteCode` is not registered in the main app's in-process activity list, DTS
uses the profile to route the work to the sandbox image when the orchestrator calls it.

### Worker profile configuration reference

The table below lists each `SandboxWorkerProfileOptions` setting, what it controls, its
accepted values, and its default.

| Setting | What it controls | Accepted values | Default |
| --- | --- | --- | --- |
| `Image.ImageRef` | The container image that holds your activity implementations. | A full OCI image reference, by tag (`myregistry.azurecr.io/workers/hello:1.0`) or digest (`myregistry.azurecr.io/workers/hello@sha256:...`). | *Required* |
| `Image.ManagedIdentityClientId` | The client ID of the user-assigned managed identity DTS uses to **pull the worker image** from your registry. This identity needs the **AcrPull** role on the registry. | A user-assigned managed identity client ID (GUID). Must be attached to the scheduler. | *Required* |
| `SchedulerManagedIdentityClientId` | The client ID of the user-assigned managed identity the **sandbox worker uses to connect back to DTS**, and that the activity code runs as when calling other services. | A user-assigned managed identity client ID (GUID). Must be attached to the scheduler. Can be the same identity as the image-pull identity or a different one. | *Required* |
| `Cpu` | CPU quantity declared for each sandbox. | A positive CPU quantity, expressed in millicores (`500m`, `1000m`) or whole/fractional cores (`2`, `0.5`). | `1000m` (1 vCPU) |
| `Memory` | Memory quantity declared for each sandbox. | A positive memory quantity, such as `256Mi`, `1Gi`, or a bare number interpreted as MiB (`2048`). | `2048Mi` |
| `MaxConcurrentActivities` | How many activities a single sandbox worker instance processes concurrently. | An integer greater than `0`. There is no enforced upper bound; size it to what your activity and resource shape can handle. | `100` |
| `EnvironmentVariables` | Customer environment variables injected into the sandbox at runtime. | A map of string keys to string values. | Empty |
| *(profile id)* | Friendly profile id that groups the image, resources, and activities for monitoring and reuse. | A non-empty string, unique across your declared profiles. | `default` |
| `AddActivity` | The activity names this profile offloads to the sandbox. | One or more activity names. At least one is required; an activity can belong to only one profile. | *Required* |

> [!NOTE]
> CPU and memory must be positive resource quantities. The platform may apply additional
> per-preview ceilings on the total CPU and memory a sandbox can request. Check your
> private preview onboarding details for the current limits.

## Configure the scheduler identity for image pull

To start a sandbox, DTS pulls your worker image from your container registry on your
behalf. It does this using a **user-assigned managed identity** attached to the scheduler.
That identity must be granted the **AcrPull** role on the Azure Container Registry that
hosts your worker image, and the scheduler must have the identity attached.

> [!IMPORTANT]
> Only **user-assigned** managed identities are supported. System-assigned managed
> identities are not supported at this time.

The worker profile distinguishes two identities, and you can use the same identity for
both or split them:

- **Image-pull identity** (`options.Image.ManagedIdentityClientId`): the identity DTS
  uses to **pull the worker image** from your registry. This identity needs the **AcrPull**
  role on the registry.
- **Worker/scheduler identity** (`options.SchedulerManagedIdentityClientId`): the
  identity the **sandbox worker uses to connect back to Durable Task Scheduler**, and the
  identity your activity code runs as when it calls other services (for example, Storage,
  Key Vault, or a database). Grant this identity whatever roles your activity code needs on
  those downstream services.

Both identities must be attached to the scheduler. Using two separate identities lets you
scope image-pull permissions narrowly while granting your activity code only the
downstream permissions it needs.

### 1. Grant the identity the AcrPull role on your registry

Assign the **AcrPull** role to the **image-pull** user-assigned managed identity, scoped
to your registry:

```bash
az role assignment create \
  --assignee "<image-pull-identity-principal-id>" \
  --role "AcrPull" \
  --scope "/subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.ContainerRegistry/registries/<registry-name>"
```

Without this role assignment, DTS cannot pull the worker image and the sandbox will fail
to start. If your activity code calls other Azure services, grant the **worker/scheduler**
identity the roles it needs on those services as well.

### 2. Attach the identity to the scheduler

The scheduler must have the user-assigned identity attached. The
[`durabletask` Azure CLI extension](https://learn.microsoft.com/cli/azure/durabletask)
provides identity commands that handle this for you. Install it with
`az extension add --name durabletask` if you haven't already.

Attach the identity to an existing scheduler:

```bash
az durabletask scheduler identity assign \
  --resource-group "<resource-group>" \
  --name "<scheduler-name>" \
  --user-assigned "/subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identity-name>"
```

To attach multiple identities (for example, separate image-pull and worker/scheduler
identities), pass several space-separated resource IDs to `--user-assigned`. Verify what's
attached with `az durabletask scheduler identity show --resource-group "<resource-group>" --name "<scheduler-name>"`.

Once the identities are attached to the scheduler (the image-pull identity with the
**AcrPull** role on your registry), reference their client IDs on the worker profile
(`options.Image.ManagedIdentityClientId` and `options.SchedulerManagedIdentityClientId`)
so DTS uses the image-pull identity to pull the image and the worker/scheduler identity for
the sandbox worker to connect back to DTS and call downstream services.

## Step 2: Build the worker image

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

`UseSandboxWorker()` is the key line. It signals that this worker runs in DTS-managed
compute. The sandbox worker does **not** need to configure the DTS endpoint, task hub,
profile id, or credentials; DTS injects the runtime settings when it starts the
container.

The activity implementations themselves are standard Durable Task activities. There's
nothing special about the activity code. It can call a runtime with different
dependencies (for example, Python and pandas) while running in an isolated container
instead of in your main app's process.

Package the image like any containerized service, including whatever runtimes and native
tools the activity needs. Push it to your container registry (for example, Azure
Container Registry) and reference the image in the worker profile's `Image.ImageRef`
option. The image-pull identity you set in `Image.ManagedIdentityClientId` must have the
**AcrPull** role on that registry.

## Step 3: View logs in the DTS dashboard

Once your sandbox activities are running, you can view their execution logs directly in
the Durable Task Scheduler dashboard. The dashboard shows real-time output from your
managed workers, including stdout, stderr, and activity lifecycle events, giving you full
visibility into what's happening inside the sandbox without configuring external log
sinks or building your own observability pipeline.

## Next steps

- [Configure the scheduler identity for image pull](#configure-the-scheduler-identity-for-image-pull)
- [End-to-end .NET sample](../samples/dotnet)
- [Python guide](./python.md)
- [Back to overview](./README.md)
