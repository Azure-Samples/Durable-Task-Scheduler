# Saga / compensating transactions (Go)

A travel booking reserves a **flight → hotel → rental car**. If a step fails,
the orchestration undoes the earlier bookings in reverse order. This is called
compensation. In the demo, the car booking for Tokyo fails, so the hotel and
flight are cancelled.

All bookings are simulations. Confirmation IDs come from a client request ID,
so repeated calls return the same IDs (idempotency keys).

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./saga`.

## Expected output

The JSON result has `status: "failed"`, `destination: "Tokyo"`, and
`error: "No rental cars available in Tokyo"`. The compensation list has the
**hotel, then the flight**, both with `status: "cancelled"`.

Cancellation activities retry up to **three times** within ten seconds. A saga
cannot guarantee that every undo step succeeds. In production, use operations
that are safe to repeat and plan a recovery process. Failed compensation is
never reported as success.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Booking order and reverse compensation |
| [activities.go](activities.go) | Simulated bookings |
| [compensation.go](compensation.go) | Cancellations, retry policy, and failure handling |
| [client.go](client.go) | Starts one trip |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks a successful booking, a
failure at each step, and a hotel cancellation that fails three times.
