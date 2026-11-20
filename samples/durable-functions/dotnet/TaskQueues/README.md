# Task queues (Durable Functions .NET isolated)

The HTTP starter tags `ProcessOrder` with `DTS_TaskQueue=orders`.
`RecordOrder` inherits `orders`; `CallPartner` and the `ValidateOrder` child
override the queue with `partner-api`. The child's activity inherits
`partner-api`. All activities return simulated results without external calls.

## Prerequisites

- .NET 10 SDK, Azure Functions Core Tools v4 with .NET 10 support, and Azurite.
- An Azure scheduler and dedicated task hub with the `orders` and
  `partner-api` queues. Complete the
  [shared task queue setup](../../../../docs/task-queues.md) before starting
  tagged work.
- An identity with Durable Task Data Contributor access.

`host.json` disables work-item filtering. Keep all functions in this app;
do not run unrelated workers against this hub.

## Run

From this directory:

```bash
cp local.settings.json.example local.settings.json
```

For Azure, set `TASKHUB_NAME` to your hub and `DTS_CONNECTION_STRING` to
`Endpoint=https://<scheduler-host>;Authentication=DefaultAzure` in the copied
settings. For a deployed app, use `Authentication=ManagedIdentity` and assign
the data-plane role to the app identity. The Functions host supplies the hub
name separately.

Start Azurite in a separate terminal, then:

```bash
func start
curl -X POST http://localhost:7071/api/orders/order-001
```

The example settings connect to the emulator on `localhost:8080`. Start the
emulator first if using them. Emulator execution does not verify Azure
queue provisioning or rate enforcement.

## Expected output

The POST returns HTTP 202 with the instance ID and management URLs. Poll
`StatusQueryGetUri` from the response until `runtimeStatus` is `Completed`.
Stop polling on `Failed` or `Terminated`, or after two minutes.
The completed `output` is:

```text
Recorded order-001; Partner accepted order-001; Validated order-001
```

Functions require a function key when deployed; local Core Tools disables key
authorization by default. Inspect tags in the scheduler dashboard. Rates
measure work-item handouts, not completed orders or partner API requests.

`Functions.cs` contains the starter, orchestrations, and activities;
`host.json` configures the unfiltered Azure Managed provider.
