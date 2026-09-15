# Testing Go workflows

Process an order through validation, payment, and shipping. The same business
workflow runs with durable activities in the application and local steps in
unit tests. Money uses integer cents to avoid floating-point rounding.

## Code map

Start with `processOrder` in [workflow.go](workflow.go).

| File | Responsibility |
| --- | --- |
| [workflow.go](workflow.go) | Order types, business workflow, and durable activity adapter |
| [activities.go](activities.go) | Validation and simulated payment/shipping operations |
| [worker.go](worker.go) | Register the orchestration and activities |
| [client.go](client.go) | Start the worker, submit one order, and print its result |
| [main.go](main.go) | CLI entrypoint |
| [workflow_test.go](workflow_test.go) | Offline business-logic and failure-path tests |
| [integration_test.go](integration_test.go) | Real DTS success/failure verification |

## Prerequisites

- Go 1.25 or later.
- No services for unit tests.
- The DTS emulator or an authorized live task hub for the demo and integration
  tests; see the [shared setup](../README.md).

## Run the demo

From this sample directory:

```bash
go run .
```

The demo submits one order and prints its completed result:

```json
{
  "paymentId": "PAY-2000",
  "trackingId": "TRACK-ALICE-1",
  "totalCents": 2000,
  "status": "completed"
}
```

The worker and client shut down afterward. `DTS_CONNECTION_STRING` selects the
backend without code changes. Payment and shipping are simulations, not external
service calls.

## Run tests

Offline tests verify activity order, exact results, input validation, overflow
protection, and propagation of payment/shipping failures:

```bash
go test -v .
```

The Go beta has no public in-memory orchestration backend. The local adapter
tests business logic, not durable replay, persistence, or transport.

With a configured DTS backend, run the integration test:

```bash
DTS_SAMPLES_E2E=1 go test -v -run '^TestIntegration$' .
```

It uses the registered production workflow to process two valid and three invalid
orders, checking exact outputs and persisted failure details. Failed instances
are intentional and remain visible in the dashboard. Verification logic lives
in test files, not in the demo.
