# On-demand Sandboxes — Python guide

> **Status:** Private preview · [Back to overview](./README.md)

This guide walks through using On-demand Sandboxes with the **Python** Durable Task SDK.
Make sure you've reviewed the [prerequisites](./README.md#prerequisites) first.

On-demand Sandboxes use a two-part model: a **sandbox worker profile** (the *declarer
app*) that tells DTS which activities to offload, and a **worker image** that contains
those activity implementations. Your orchestrator still calls activities the same way it
always has—the decision to run one in a sandbox lives entirely in the profile
configuration.

## Install the SDK

The on-demand sandbox APIs ship under the `durabletask.azuremanaged.preview.sandboxes`
namespace. Install the Durable Task packages:

```bash
pip install durabletask==1.6.0 durabletask-azuremanaged==1.6.0
```

## Step 1 — Declare a sandbox worker profile

The declarer app uses a decorated profile class to declare the remote worker image and
activity ownership, then enables sandbox activities on the DTS client. The profile sets
the image, the managed identities DTS needs to pull the image and start the sandbox, the
resource shape, concurrency, any customer environment variables, and the activity names
to offload with `options.add_activity(...)`.

```python
import os

from azure.identity import DefaultAzureCredential

from durabletask import client, task
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.azuremanaged.preview.sandboxes import (
    SandboxActivitiesClient,
    SandboxWorkerProfile,
    sandbox_worker_profile,
)
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

REMOTE_HELLO = "remote_hello"

endpoint = os.environ["DTS_ENDPOINT"]
taskhub_name = os.environ["DTS_TASK_HUB"]
worker_profile_id = os.getenv("DTS_WORKER_PROFILE_ID", "default")
container_image = os.environ["DTS_SANDBOX_CONTAINER_IMAGE"]


def hello_orchestrator(ctx: task.OrchestrationContext, name: str):
    """Orchestrator that calls an activity executed by the remote sandbox worker."""
    return (yield ctx.call_activity(REMOTE_HELLO, input=name))


@sandbox_worker_profile(worker_profile_id)
class RemoteWorkerProfile(SandboxWorkerProfile):
    def configure(self, options) -> None:
        options.image.image_ref = container_image
        options.image.managed_identity_client_id = os.environ[
            "DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID"]
        options.scheduler_managed_identity_client_id = os.environ[
            "DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID"]
        options.cpu = "1000m"
        options.memory = "2048Mi"
        options.max_concurrent_activities = 1
        options.environment_variables["SAMPLE_MARKER"] = "python-sample-marker"
        options.add_activity(REMOTE_HELLO)


credential = DefaultAzureCredential()

# Declare the sandbox worker profile with DTS so it can route the activity to a sandbox.
sandbox_client = SandboxActivitiesClient(
    host_address=endpoint,
    secure_channel=True,
    taskhub=taskhub_name,
    token_credential=credential)
sandbox_client.enable_sandbox_activities()

with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=True,
        taskhub=taskhub_name,
        token_credential=credential) as worker:
    worker.add_orchestrator(hello_orchestrator)
    worker.use_work_item_filters()
    worker.start()

    durable_client = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=True,
        taskhub=taskhub_name,
        token_credential=credential)
    instance_id = durable_client.schedule_new_orchestration(
        hello_orchestrator, input="on-demand sandbox Python")
    state = durable_client.wait_for_orchestration_completion(instance_id, timeout=300)
    print(state.serialized_output if state else "no result")
```

`enable_sandbox_activities()` is the key call—it registers the declared profiles with DTS
so it can route those activities to the sandbox image. `use_work_item_filters()` keeps
sandbox activities from being dispatched to this in-process worker.

> [!IMPORTANT]
> The managed identities referenced by `options.image.managed_identity_client_id` and
> `options.scheduler_managed_identity_client_id` must both be attached to the scheduler.
> The image-pull identity must have the **AcrPull** role on your container registry, and
> the worker/scheduler identity must have whatever roles your activity code needs on the
> downstream services it calls (you can use the same identity for both or split them). See
> [Configure the scheduler identity for image pull](./README.md#configure-the-scheduler-identity-for-image-pull).

For the meaning, accepted values, and defaults of each profile option, see the
[worker profile configuration reference](./README.md#worker-profile-configuration-reference).
In short: `image.image_ref` is the image with your activity implementations;
`image.managed_identity_client_id` is the managed identity DTS uses to **pull the worker
image** from your registry (needs **AcrPull**), while
`scheduler_managed_identity_client_id` is the managed identity the **sandbox worker uses
to connect back to DTS** and that the activity code runs as when it calls other services;
`cpu` / `memory` set the per-sandbox resource shape; `max_concurrent_activities` sets
concurrency; `environment_variables` injects customer environment variables; and
`add_activity(...)` selects the activities to offload (only added activities run in
DTS-managed isolated compute; everything else stays in-process).

The orchestrator call site doesn't change—it calls `REMOTE_HELLO` the same way it would
call any activity, and DTS routes it to the sandbox.

## Step 2 — Build the worker image

The worker image runs `SandboxWorker()`, registers the activity implementations it owns,
and starts. The sandbox worker does **not** configure the DTS endpoint, task hub, profile
id, or credentials—`SandboxWorker()` reads the runtime settings (such as `DTS_ENDPOINT`,
`DTS_TASK_HUB`, `DTS_WORKER_PROFILE_ID`, and `DTS_SANDBOX_ID`) from environment variables
that DTS injects when it starts the container.

```python
import os
import threading

from durabletask import task
from durabletask.azuremanaged.preview.sandboxes import SandboxWorker

REMOTE_HELLO = "remote_hello"


def _remote_hello(ctx: task.ActivityContext, name: str) -> str:
    """Activity function that runs inside the on-demand sandbox worker container."""
    sandbox_id = os.getenv("DTS_SANDBOX_ID", "unknown-sandbox")
    return f"Hello {name} from Python on-demand sandbox worker {sandbox_id}!"


# The registered activity name must match the name declared in the worker profile.
_remote_hello.__name__ = REMOTE_HELLO

with SandboxWorker() as worker:
    worker.add_activity(_remote_hello)
    worker.start()
    print("Python on-demand sandbox remote worker is running.")
    try:
        threading.Event().wait()
    except KeyboardInterrupt:
        pass
```

Keep the activity name constant (here, `REMOTE_HELLO`) in a small shared module so the
declarer app and the remote worker stay in sync. When the worker connects, it reports its
registered activity names, and DTS validates they match the declaration before
advertising worker capacity.

Build and push the image with a `Containerfile`/`Dockerfile` that installs the SDK and
your activity's dependencies, then copies in the worker entry point:

```dockerfile
# syntax=docker/dockerfile:1.7
FROM python:3.12-slim AS runtime
WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
ENV GRPC_DEFAULT_SSL_ROOTS_FILE_PATH=/etc/ssl/certs/ca-certificates.crt

# Install the Durable Task SDK (with the sandboxes extension), plus your
# activity dependencies.
RUN pip install --no-cache-dir durabletask==1.6.0 durabletask-azuremanaged==1.6.0

COPY remote_worker.py /app/remote_worker.py
COPY activities.py /app/activities.py

ENTRYPOINT ["python", "/app/remote_worker.py"]
```

```bash
docker build -f Containerfile -t <container image reference> .
docker push <container image reference>
```

Then set the image reference on the declarer profile (for example, via the
`DTS_SANDBOX_CONTAINER_IMAGE` environment variable). DTS pulls the image using the
image-pull managed identity you configured on the profile, which must have the
**AcrPull** role on your registry.

## Step 3 — View logs in the DTS dashboard

Once your sandbox activities are running, you can view their execution logs directly in
the Durable Task Scheduler dashboard. See
[View logs in the DTS dashboard](./README.md#view-logs-in-the-dts-dashboard) in the
overview for details.

## Next steps

- [Worker profile configuration reference](./README.md#worker-profile-configuration-reference)
- [Configure the scheduler identity for image pull](./README.md#configure-the-scheduler-identity-for-image-pull)
- [End-to-end Python sample](../samples/python)
- [.NET guide](./dotnet.md)
- [Back to overview](./README.md)
