# Saga / compensating transactions — Go

A travel-booking saga reserves a **flight → hotel → rental car**. A failed
booking compensates successful earlier bookings in reverse order. The demo
shows one Tokyo trip whose car booking is deliberately rejected, causing the
hotel and flight to be cancelled.

All booking and cancellation operations are explicitly **simulations**. No
provider is contacted and no money is charged. Confirmation IDs are stable
derivatives of a client-created request ID, rather than wall-clock timestamps.
They illustrate idempotency keys, not a real persistent booking store.

## Prerequisites

- Go **1.25 or newer**, Docker, and a running Durable Task Scheduler emulator.
- Follow [shared emulator and live Azure setup](../README.md).
- Uses the shared Go module and Durable Task Go SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./saga`.
One process starts worker and client, runs the trip, and prints its rollback
result. It does not run a scenario matrix or inspect history. Normal execution
takes a few seconds. The outer `-timeout` defaults to two minutes.

## Expected results

The JSON result has `status: "failed"`, `destination: "Tokyo"`, and
`error: "No rental cars available in Tokyo"`. Its compensation list contains
the **hotel, then flight**, both with `status: "cancelled"`. Unique instance and
confirmation IDs identify the simulated trip. Successful rollback is a completed
**business failure**, not a successful booking. Unexpected activity/SDK errors
are propagated after attempting compensation.

Cancellation activities have a **three-attempt** durable retry policy with
100 ms initial delay, exponential backoff, and a ten-second retry budget. The
integration suite exercises exhausted retries; the demo's cancellations succeed
on their first attempts.

Expected failed-activity warnings may also appear. Compensation errors remain
visible in custom status and the orchestration's typed failure details. A saga
cannot guarantee an atomic rollback when providers fail; production systems
need idempotent operations and an operational/manual recovery path for this
case. The sample never hides failed compensation behind a success result.

Inspect the trip at <http://localhost:8082>. Nothing is purged.
Registrations start with `GoSaga`, with automatic worker filters.
All work settles before shutdown; error cleanup
targets only this run's own instance.

## Code map

[workflow.go](workflow.go) shows the booking sequence and reverse compensation.
[activities.go](activities.go) contains booking payloads and simulations;
[compensation.go](compensation.go) contains cancellations, retry policy, and
typed failure handling. [client.go](client.go) starts one trip,
[worker.go](worker.go) registers tasks, and [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests verify typed activity payloads, booking order, reverse compensation, early
failures, remaining compensation after an error, unexpected-error propagation,
stable fixture confirmations, and retry-evidence validation. The unit activity
invoker does not emulate SDK retries. Run the complete backend suite explicitly:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) verifies successful Paris booking,
flight/hotel/car rejection, and hotel cancellation failure. It checks exact
receipts and compensation order, the typed `FAILED` status for incomplete
compensation, **three hotel cancellation attempts**, and successful remaining
flight compensation. History API errors fail the test rather than bypassing
verification. Only this opt-in suite runs all five scenarios.
