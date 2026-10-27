# Testing Go workflows

An order goes through validation, payment, and shipping. The app runs this
logic as a durable orchestration. Unit tests run the same business logic with
local steps, without DTS. Money is stored in whole cents.

The Go beta SDK has no public in-memory orchestration backend. The unit tests
check business logic only. They do not check replay or DTS behavior. Use the
integration test for that.

## Prerequisites

- Go 1.25 or later. Unit tests need no services.
- For the demo and integration test, the emulator or an Azure task hub. See the
  [shared Go README](../README.md).

## Run the demo

From this directory:

```bash
go run .
```

Expected output:

```json
{
  "paymentId": "PAY-2000",
  "trackingId": "TRACK-ALICE-1",
  "totalCents": 2000,
  "status": "completed"
}
```

Payment and shipping are simulations.

## Run tests

```bash
go test -v .
DTS_SAMPLES_E2E=1 go test -v -run '^TestIntegration$' .
```

Unit tests check step order, results, input rules, and error handling. The
integration test runs two valid and three invalid orders through the real
orchestration. The failed instances are expected and stay in the dashboard.

## Code map

Start with `processOrder` in [workflow.go](workflow.go).

| File | Purpose |
| --- | --- |
| [workflow.go](workflow.go) | Order data, business logic, and activity calls |
| [activities.go](activities.go) | Validation and simulated payment and shipping |
| [worker.go](worker.go) | Registers the orchestration and activities |
| [client.go](client.go) | Starts the worker, submits one order, and prints the result |
| [main.go](main.go) | Command-line entry point |
| [workflow_test.go](workflow_test.go) | Offline business-logic tests |
| [integration_test.go](integration_test.go) | DTS success and failure tests |
