# Saga / compensating transactions (Go)

A travel-booking saga reserves a **flight → hotel → rental car**.
If a booking fails, the workflow undoes earlier bookings in reverse order.
These undo steps are called compensation. The demo rejects the car booking
for a Tokyo trip, then cancels the hotel and flight.

All bookings and cancellations are **simulations**. They do not contact providers
or charge money. Confirmation IDs are built from a request ID created by the
client. Repeated calls use the same IDs. This shows how idempotency keys can help
avoid duplicate bookings, but the sample has no real booking store.

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
result. It does not run all test cases or read history. Normal execution
takes a few seconds. The outer `-timeout` defaults to two minutes.

## Expected results

The JSON result has `status: "failed"`, `destination: "Tokyo"`, and
`error: "No rental cars available in Tokyo"`. Its compensation list contains
the **hotel, then flight**, both with `status: "cancelled"`. Unique instance and
confirmation IDs identify the simulated trip. The workflow finishes after it
undoes the bookings, but the trip still has a **business failure**.
Unexpected activity or SDK errors are returned after compensation is attempted.

Cancellation activities allow **three attempts** within ten seconds.
The first retry delay is 100 ms, and later delays grow. The integration tests
check what happens when all attempts fail. In the demo, each cancellation
succeeds on its first attempt.

Warnings may appear for the simulated booking failure.
Compensation errors remain visible in custom status and the failure details.
A saga cannot guarantee that every undo step succeeds when a provider fails.
Production apps need operations that are safe to repeat and a recovery process,
which may include manual work. Failed compensation is never reported as success.

View the trip at <http://localhost:8082>. Nothing is deleted.
Registrations start with `GoSaga`, with automatic worker filters.
All work finishes before shutdown. Error cleanup
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

Tests check activity data, booking order, reverse compensation, and early
failures. They also check that other undo steps continue after one fails,
errors reach the caller, and confirmation IDs stay stable.
The unit tests do not run the SDK retry process. Enable the full backend suite with:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) checks a successful Paris booking,
rejected flight, hotel, and car bookings, and a hotel cancellation failure.
It checks the results and undo order. Incomplete compensation must have `FAILED`
status. The hotel cancellation must be attempted **three times**, and the flight
must still be cancelled. History API errors fail the test.
Only this enabled integration suite runs all five cases.
