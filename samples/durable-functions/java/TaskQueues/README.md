# Task queues with Durable Functions Java

`POST /api/orders` starts `ProcessOrder` with `DTS_TaskQueue=orders`.
It calls `ValidateOrder` and the `FulfillOrder` child orchestration. The child
calls `CallPartnerApi` and `ShipOrder`. All inherit `orders` when that queue
exists.
The partner call only logs a message.

Durable Functions Java 1.9.0 uses the Java SDK's start options with input and
tags. Java does not expose activity or child-orchestration tag overrides.
The partner activity stays in `orders`; this sample does not use `partner-api`.

## Prerequisites

- JDK 21, Maven, and Azure Functions Core Tools v4.
- Azure CLI signed in with `az login`.
- An Azure scheduler and task hub. Assign Durable Task Data Contributor to the
  identity running the Functions host.
- Azurite for local Functions host storage, or an Azure Storage connection string.
  Functions still needs `AzureWebJobsStorage` when orchestration state uses DTS.
- Bash, curl, and jq for `demo.sh`.
- Follow [shared task queue setup](../../../../docs/task-queues.md) to provision
  `orders` at 20 work-item handouts per second and `partner-api` at 5.
  Wait for provisioning to succeed before running. Only `orders` is used here.

Actual queue throttling requires Azure. The emulator can run the workflow but
cannot validate Azure task queue limits. Missing or deleted queues fall back to
ordinary task hub routing without the requested queue's rate limit.

## Configure and run

Run from this directory:

```bash
cp local.settings.json.example local.settings.json
```

Edit `local.settings.json` with your endpoint and hub. Set `TASK_HUB_NAME` to the
same hub as the connection string. Start Azurite separately if using
`UseDevelopmentStorage=true`.

For the local emulator, set `TASK_HUB_NAME` to `default` and
`DURABLE_TASK_SCHEDULER_CONNECTION_STRING` to
`Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`.
Use a different hub only if it exists in the emulator configuration.

```bash
mvn package
mvn azure-functions:run
```

In another terminal:

```bash
bash demo.sh
```

The script starts one order, then polls for at most three minutes, with bounded
HTTP request timeouts. It fails on failed, terminated, unexpected, or missing
states and on timeout. An in-flight HTTP request can extend the polling deadline
by up to ten seconds. The instance remains persisted after a polling timeout.

To start an order manually and inspect status:

```bash
curl --fail --max-time 15 -H 'Content-Type: text/plain' \
  -d 'order-2' http://localhost:7071/api/orders
curl --max-time 10 http://localhost:7071/api/orders/INSTANCE_ID
```

The starter returns HTTP 202 with the instance ID and standard Durable Functions
management URLs. The custom GET route returns 202 while pending, running, or
suspended, 200 with output when completed, 500 when failed or terminated, and
404 for an unknown instance. The Functions extension manages the shared client;
do not close it from an invocation. The Functions host owns and stops its
worker; stop the host with Ctrl+C.

Authorization is `FUNCTION`, enforced when hosted in Azure. Set `BASE_URL` and
`FUNCTION_KEY` when running `demo.sh` against a deployed app. Do not commit
`local.settings.json` or function keys.

## Worker eligibility

`host.json` sets `extensions.durableTask.storageProvider.workItemFilteringEnabled`
to `false`. Keep the connection unfiltered. Wildcard filters are also ineligible
for defined task queues. Use a dedicated hub or only identical workers registering
every task in this sample.

`host.json` requires extension bundle 4.41.2 or later within v4.
[Bundle 4.41.2](https://github.com/Azure/azure-functions-extension-bundles/releases/tag/4.41.2)
includes Durable Functions extension 3.15.3 and AzureManaged provider 1.10.2.
The [released gRPC server](https://github.com/Azure/azure-functions-durable-extension/blob/v3.15.3/src/WebJobs.Extensions.DurableTask/TaskHubGrpcServer.cs)
passes start tags into the orchestration's execution-started event.
The [released filtering sample](https://github.com/Azure/azure-functions-durable-extension/tree/v3.15.3/samples/workitem-filters)
documents `workItemFilteringEnabled=false` as disabling filtering.

## Expected output

The host logs validation and the simulated partner call. The script prints:

```text
Started <instance-id> with DTS_TaskQueue=orders
{
  "instanceId": "<instance-id>",
  "runtimeStatus": "COMPLETED",
  "output": "Shipped order-1"
}
```

Inspect orchestration tags and status in the scheduler dashboard.
The sustained queue rate includes orchestration turns, activity deliveries, and
redeliveries. It does not guarantee 20 completed orders or API requests per second.
Short bursts can exceed the sustained rate.

See the [task queue documentation](https://learn.microsoft.com/azure/durable-task/scheduler/durable-task-scheduler-task-queues)
for routing and SDK support.
