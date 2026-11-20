# Task queues with the Java Durable Task SDK

This order-processing sample starts ten `ProcessOrder` instances with
`DTS_TaskQueue=orders`. Each calls `ValidateOrder` and the `FulfillOrder` child
orchestration. The child calls `CallPartnerApi` and `ShipOrder`. All inherit
`orders` when that queue exists. The partner call only prints a message.

Java SDK 1.9.0 supports orchestration start tags, available since 1.6.0.
It does not expose activity or child-orchestration tag overrides. This sample
does not route the partner activity to `partner-api`.

## Prerequisites

- JDK 21.
- An Azure scheduler and task hub with the Durable Task Data Contributor role
  assigned to the identity running the sample.
- Azure CLI, signed in with `az login` for `Authentication=DefaultAzure`.
- Follow [shared task queue setup](../../../../docs/task-queues.md) to provision
  `orders` at 20 work-item handouts per second. Wait for provisioning to
  succeed before running. This sample does not use `partner-api`.

Actual queue throttling requires Azure. The emulator can exercise the workflow,
but does not validate the Azure queue rate. If the queue is missing or deleted,
the requested queue falls back to ordinary task hub routing without its rate limit.

## Run

Run from this directory:

```bash
export DURABLE_TASK_CONNECTION_STRING='Endpoint=https://YOUR-SCHEDULER-ENDPOINT;TaskHub=YOUR-HUB;Authentication=DefaultAzure'
bash ./gradlew --no-daemon build
bash ./gradlew --no-daemon run --args='10'
```

For the local emulator, set the connection string below and run the same
Gradle commands:

```bash
export DURABLE_TASK_CONNECTION_STRING='Endpoint=http://localhost:8080;TaskHub=default;Authentication=None'
```

The order count is optional and must be between 1 and 100. The batch has a
three-minute completion deadline and exits with an error on timeout, missing
instances, or any terminal state other than `COMPLETED`. Worker and client
resources are closed on both success and failure. A timeout leaves persisted
instances in the scheduler; inspect them in the dashboard before rerunning.

The worker is unfiltered and registers every task used by the workflow.
Use a dedicated task hub or only identical workers in this hub. A filtered
connection, including wildcard filters, cannot receive work from a defined queue.

## Expected output

Instance IDs and interleaving vary:

```text
Started <instance-id> with DTS_TaskQueue=orders
Validated order-1
Simulated partner call for order-1
Shipped order-1
...
Completed 10 orders.
```

Inspect the orchestration tags and status in the scheduler dashboard.
The configured rate counts orchestration turns and activity deliveries together,
including redeliveries. It does not promise 20 completed orders or partner API
requests per second. Short bursts can exceed the sustained rate.

## References

- [Task queues](https://learn.microsoft.com/azure/durable-task/scheduler/durable-task-scheduler-task-queues)
- [Java start options](https://github.com/microsoft/durabletask-java/blob/v1.9.0/client/src/main/java/com/microsoft/durabletask/NewOrchestrationInstanceOptions.java)
