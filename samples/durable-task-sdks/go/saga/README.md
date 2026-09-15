# Saga / compensating transactions — Go

A travel-booking saga reserves a **flight → hotel → rental car**. A failed
booking compensates successful earlier bookings in reverse order. This preserves
the Python sample's Paris success and Tokyo car-failure scenarios, and adds
verification of earlier failures and exhausted compensation retries.

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
One process starts worker and client and verifies five bounded scenarios. Normal
execution takes under a minute. The outer `-timeout` defaults to two minutes.

## Expected results

| Scenario | Business result | Compensation order | Orchestration status |
|---|---|---|---|
| Paris, five nights | success | none | COMPLETED |
| Tokyo, no rental car | failed | hotel, flight | COMPLETED |
| Paris, zero hotel nights | failed | flight | COMPLETED |
| Nowhere, no flight | failed | none | COMPLETED |
| Tokyo, car failure plus hotel cancellation outage | compensation_failed | hotel fails; flight still cancelled | **FAILED** |

Each JSON result includes unique instance/confirmation IDs, exact booking
receipts, and ordered compensation results. Successful rollback is a completed
**business failure**, not a successful booking. Unexpected activity/SDK errors
are propagated after attempting compensation.

Cancellation activities have a **three-attempt** durable retry policy with
100 ms initial delay, exponential backoff, and a ten-second retry budget. The
last scenario verifies actual history contains **three hotel cancellation
attempts and one successful flight cancellation**. An unavailable history API or
an unexpected failure is an error, not a skipped check.

The final line appears only after all expected results and the deliberate
runtime failure are verified:

```text
SAMPLE_OK saga
```

Expected failed-activity warnings may also appear. Compensation errors remain
visible in custom status and the orchestration's typed failure details. A saga
cannot guarantee an atomic rollback when providers fail; production systems
need idempotent operations and an operational/manual recovery path for this
case. The sample never hides failed compensation behind a success result.

Inspect all instances, including the deliberate `FAILED` instance, at
<http://localhost:8082>. Nothing is purged. Registrations start with `GoSaga`, with
automatic worker filters. All work settles before shutdown; error cleanup
targets only this run's own instance.

## Unit tests

```bash
go test -mod=readonly .
```

Tests verify typed activity payloads, booking order, reverse compensation, early
failures, remaining compensation after an error, unexpected-error propagation,
stable fixture confirmations, and retry-evidence validation. The unit activity
invoker does not emulate SDK retries; the runnable client verifies those against
the actual scheduler.
