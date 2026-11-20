# Task queues (JavaScript SDK)

`processOrder` starts with `DTS_TaskQueue=orders`. When the queues exist, its
`recordOrder` activity inherits `orders`, while `callPartner` and the
`validateOrder` child explicitly use `partner-api`. The child's activity
inherits `partner-api`. Activities return simulated responses and make no
external API calls.

## Prerequisites

- Node.js 22 or later.
- An Azure scheduler and a dedicated task hub with `orders` and `partner-api`.
  Complete the [shared task queue setup](../../../../docs/task-queues.md).
- An identity with Durable Task Data Contributor access.

This SDK worker connects without work-item filters. Use only this worker
(or identical replicas) in the hub.

## Run

From this directory:

```bash
npm ci
export DTS_CONNECTION_STRING="Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure"
npm start -- order-001
```

Without the variable, the sample connects to the emulator at `localhost:8080`
in hub `default`. Emulator execution does not verify Azure queue provisioning
or rate enforcement.

## Expected output

```text
Started <instance-id> with DTS_TaskQueue=orders
Recorded order-001; Partner accepted order-001; Validated order-001
```

The client checks completion and output, waits at most 120 seconds, then closes
the worker and client. A timeout does not terminate the persisted instance.
Inspect status and tags in the scheduler dashboard.

`main.mjs` configures the worker and tagged start; `workflows.mjs` contains
the inherited and overridden calls.

## Tests

```bash
npm test
```

Offline tests check the activity and child options, workflow output, and
failure propagation. They do not test scheduler routing or throttling.
