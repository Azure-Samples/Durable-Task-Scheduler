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

The on-demand sandbox APIs ship in a preview package namespace,
`durabletask.azuremanaged.preview.on_demand_sandbox`. Install the Durable Task packages:

```bash
pip install durabletask==1.6.0 durabletask.azuremanaged==1.6.0
```

> [!NOTE]
> The on-demand sandbox APIs are available starting in `durabletask==1.6.0` and
> `durabletask.azuremanaged==1.6.0`, under the
> `durabletask.azuremanaged.preview.on_demand_sandbox` namespace.

## Step 1 — Declare a sandbox worker profile

The declarer app uses a decorated profile class to declare the remote worker image and
activity ownership, then enables on-demand sandbox activities on the DTS client. The
profile sets the image, the managed identities DTS needs to pull the image and start the
sandbox, the resource shape, concurrency, any customer environment variables, and the
activity names to offload with `options.add_activity(...)`.

```python
import os

from azure.identity import DefaultAzureCredential

from durabletask import client, task
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.azuremanaged.preview.on_demand_sandbox import (
    OnDemandSandboxActivitiesClient,
    OnDemandSandboxWorkerProfile,
    on_demand_sandbox_worker_profile,
)
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

REMOTE_HELLO = "remote_hello"

endpoint = os.environ["DTS_ENDPOINT"]
taskhub_name = os.environ["DTS_TASK_HUB"]
worker_profile_id = os.getenv("DTS_WORKER_PROFILE_ID", "default")
container_image = os.environ["DTS_ON_DEMAND_SANDBOX_CONTAINER_IMAGE"]


def hello_orchestrator(ctx: task.OrchestrationContext, name: str):
    """Orchestrator that calls an activity executed by the remote sandbox worker."""
    return (yield ctx.call_activity(REMOTE_HELLO, input=name))


@on_demand_sandbox_worker_profile(worker_profile_id)
class RemoteWorkerProfile(OnDemandSandboxWorkerProfile):
    def configure(self, options) -> None:
        options.container_image = container_image
        options.image_pull_managed_identity_client_id = os.environ[
            "DTS_ON_DEMAND_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID"]
        options.scheduler_managed_identity_client_id = os.environ[
            "DTS_ON_DEMAND_SANDBOX_SCHEDULER_UMI_CLIENT_ID"]
        options.cpu = "1000m"
        options.memory = "2048Mi"
        options.max_concurrent_activities = 1
        options.environment_variables["SAMPLE_MARKER"] = "python-sample-marker"
        options.add_activity(REMOTE_HELLO)


credential = DefaultAzureCredential()

# Register the on-demand sandbox activity metadata with DTS.
sandbox_client = OnDemandSandboxActivitiesClient(
    host_address=endpoint,
    secure_channel=True,
    taskhub=taskhub_name,
    token_credential=credential)
sandbox_client.enable_on_demand_sandbox_activities()

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

`enable_on_demand_sandbox_activities()` is the key call—it registers the declared
profiles with DTS so it can route those activities to the sandbox image.

For the meaning, accepted values, and defaults of each profile option, see the
[worker profile configuration reference](./README.md#worker-profile-configuration-reference).
In short: `container_image` is the image with your activity implementations;
`image_pull_managed_identity_client_id` / `scheduler_managed_identity_client_id` are the
managed identity client IDs DTS uses to pull the image and start the sandbox; `cpu` /
`memory` set the per-sandbox resource shape; `max_concurrent_activities` sets concurrency;
`environment_variables` injects customer environment variables; and `add_activity(...)`
selects the activities to offload (only added activities run in DTS-managed isolated
compute; everything else stays in-process).

The orchestrator call site doesn't change—it calls `REMOTE_HELLO` the same way it would
call any activity, and DTS routes it to the sandbox.

## Step 2 — Build the worker image

The worker image runs `OnDemandSandboxWorker()`, registers the activity implementations
it owns, and starts. The sandbox worker does **not** configure the DTS endpoint, task
hub, profile id, or credentials—`OnDemandSandboxWorker()` reads the runtime settings
(`DTS_ENDPOINT`, `DTS_TASK_HUB`, `DTS_WORKER_PROFILE_ID`,
`DTS_ON_DEMAND_SANDBOX_MAX_ACTIVITIES`, `DTS_SANDBOX_ID`, and related values) from
environment variables that DTS injects when it starts the container.

```python
import os
import threading

from durabletask import task
from durabletask.azuremanaged.preview.on_demand_sandbox import OnDemandSandboxWorker

REMOTE_HELLO = "remote_hello"


def _remote_hello(ctx: task.ActivityContext, name: str) -> str:
    """Activity function that runs inside the on-demand sandbox worker container."""
    sandbox_id = os.getenv("DTS_SANDBOX_ID", "unknown-sandbox")
    return f"Hello {name} from Python on-demand sandbox worker {sandbox_id}!"


# The registered activity name must match the name declared in the worker profile.
_remote_hello.__name__ = REMOTE_HELLO

with OnDemandSandboxWorker() as worker:
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
FROM python:3.12-slim AS runtime
WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
ENV GRPC_DEFAULT_SSL_ROOTS_FILE_PATH=/etc/ssl/certs/ca-certificates.crt

# Install the Durable Task SDKs and your activity dependencies.
RUN pip install --no-cache-dir durabletask==1.6.0 durabletask.azuremanaged==1.6.0

COPY remote_worker.py /app/remote_worker.py
COPY activity_names.py /app/activity_names.py

EXPOSE 8080
ENTRYPOINT ["python", "/app/remote_worker.py"]
```

```bash
docker build -f Containerfile -t <container image reference> .
docker push <container image reference>
```

Then set the image reference on the declarer profile (for example, via the
`DTS_ON_DEMAND_SANDBOX_CONTAINER_IMAGE` environment variable). During private preview the
image must be publicly pullable by the sandbox platform.

## Step 3 — View logs in the DTS dashboard

Once your sandbox activities are running, you can view their execution logs directly in
the Durable Task Scheduler dashboard. See
[View logs in the DTS dashboard](./README.md#view-logs-in-the-dts-dashboard) in the
overview for details.

## Next steps

- [Worker profile configuration reference](./README.md#worker-profile-configuration-reference)
- [End-to-end sample (`examples/on_demand_sandbox`)](https://github.com/microsoft/durabletask-python/tree/main/examples/on_demand_sandbox)
- [.NET guide](./dotnet.md)
- [Back to overview](./README.md)
