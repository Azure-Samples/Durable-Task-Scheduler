# Task queues (TypeScript SDK)

This typed order workflow starts with `DTS_TaskQueue=orders`. When the queues
exist, its `recordOrder` activity inherits `orders`, while `callPartner` and
the `validateOrder` child override it with `partner-api`. The child's activity
inherits `partner-api`. Activities return simulated results without external
API calls.

## Prerequisites

- Node.js 22 or later.
- An Azure scheduler and dedicated task hub with `orders` and `partner-api`.
  Complete the [shared task queue setup](../../../../docs/task-queues.md).
- An identity with Durable Task Data Contributor access.

The worker connects without work-item filters. Run only this worker
(or identical replicas) in its hub.

## Run

From this directory:

```bash
npm ci
npm run build
export DTS_CONNECTION_STRING="Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure"
npm start -- order-001
```

Without the variable, the worker and client use the emulator at `localhost:8080`
in hub `default`. Emulator execution does not verify Azure queue provisioning
or throttling.

## Expected output

```text
Started <instance-id> with DTS_TaskQueue=orders
Recorded order-001; Partner accepted order-001; Validated order-001
```

The client waits at most 120 seconds, checks status and output, and closes the
worker and client. A timeout does not terminate the persisted instance.
Inspect tags in the scheduler dashboard.

`src/main.ts` configures connections and the tagged start; `src/workflows.ts`
contains typed orchestrations and activities.

## Tests

```bash
npm test
```

This compiles with strict TypeScript checks, then tests call options, output,
and failure propagation offline. It does not test scheduler routing or rates.
