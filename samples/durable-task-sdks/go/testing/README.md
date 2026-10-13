# Testing Go workflows

Process an order through validation, payment, and shipping. The application
runs the workflow with durable activities. Unit tests use local steps to test
the same business logic. Money is stored as whole cents to avoid rounding errors.

## Code map

Start with `processOrder` in [workflow.go](workflow.go).

| File | Responsibility |
| --- | --- |
| [workflow.go](workflow.go) | Order data, business workflow, and calls to durable activities |
| [activities.go](activities.go) | Validation and simulated payment/shipping operations |
| [worker.go](worker.go) | Register the orchestration and activities |
| [client.go](client.go) | Start the worker, submit one order, and print its result |
| [main.go](main.go) | Starts the command-line program |
| [workflow_test.go](workflow_test.go) | Offline tests for business logic and errors |
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

Offline tests check activity order, results, input rules, and amounts that are
too large. They also check that payment and shipping errors reach the caller:

```bash
go test -v .
```

The Go beta has no public in-memory orchestration backend. Local tests check
business logic. They do not check replay, saved workflow state, or communication
with DTS.

With a configured DTS backend, run the integration test:

```bash
DTS_SAMPLES_E2E=1 go test -v -run '^TestIntegration$' .
```

It uses the registered production workflow to process two valid and three invalid
orders. It checks exact outputs and saved failure details. Failed instances
are intentional and remain visible in the dashboard. Verification logic lives
in test files, not in the demo.
