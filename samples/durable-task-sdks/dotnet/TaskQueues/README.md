# Task queues (.NET SDK)

Start an order workflow with `DTS_TaskQueue=orders`. When the queues exist,
`RecordOrder` inherits `orders`, while `CallPartner` and the `ValidateOrder`
child explicitly use `partner-api`. The child's `RecordOrder` activity
inherits `partner-api`. The partner activity returns a simulated response and
makes no external API call.

## Prerequisites

- .NET 10 SDK.
- For throttling, an Azure scheduler and a dedicated task hub with the
  `orders` and `partner-api` queues provisioned. Follow the
  [shared task queue setup](../../../../docs/task-queues.md) first.
- An identity with Durable Task Data Contributor access.

The worker explicitly calls `UseWorkItemFilters(null)` because the .NET SDK
automatically generates filters otherwise. Run only this sample's worker
(or identical replicas) in its task hub.

## Run

From this directory:

```bash
export DTS_CONNECTION_STRING="Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure"
dotnet run -- order-001
```

Without the environment variable, the sample connects to the emulator at
`localhost:8080`. This exercises the workflow but does not demonstrate
Azure queue provisioning or verify throttling.

## Expected output

```text
Started <instance-id> with DTS_TaskQueue=orders
Recorded order-001; Partner accepted order-001; Validated order-001
```

The process checks completion and output, waits at most two minutes, then stops
its worker and client. A timeout does not terminate the persisted orchestration.
Use the scheduler dashboard to inspect its status and tags.

`Program.cs` configures the unfiltered worker and tagged start; `Tasks.cs`
contains the inherited and overridden calls. Queue limits count work-item
handouts, not completed orders or API requests.
