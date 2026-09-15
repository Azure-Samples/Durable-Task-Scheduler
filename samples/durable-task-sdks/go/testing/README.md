# Testing Go workflows

This counterpart to the [Python testing sample](../../python/testing/) separates
order-processing logic from the durable activity adapter. It validates an order,
calculates its total, charges a simulated payment, and produces a simulated
shipment tracking ID. Money uses integer cents to avoid floating-point rounding.

## Prerequisites

- Go 1.25 or later.
- No services for unit tests.
- The DTS emulator or an authorized live task hub for integration tests; see the
  [shared setup](../README.md).

## Run

```bash
cd samples/durable-task-sdks/go/testing
go test -v .
```

Offline tests run the **same business workflow** with a local activity adapter.
They assert activity order, exact results, validation failures, overflow
protection, and propagation of payment/shipping failures.

**Unlike the Python SDK, the Go beta does not expose an in-memory testing
backend.** The local adapter is not an orchestration engine and does not verify
durable replay, persistence, or transport. Do not use internal SDK protobuf APIs
as a substitute for a public test backend.

Run the real registered orchestrator and activities on DTS:

```bash
go run .
# Or run the opt-in integration test:
DTS_SAMPLES_E2E=1 go test -v -run TestOrdersOnDTS .
```

The command starts a worker, submits two valid and three invalid orders, checks
the actual terminal status and output/failure chain of every instance, and stops
the worker. `DTS_CONNECTION_STRING` selects emulator or live DTS without code
changes.

## Expected output

```text
Verified single: go-testing-single-...
Verified multiple: go-testing-multiple-...
Verified missing-customer: go-testing-missing-customer-...
Verified empty: go-testing-empty-...
Verified invalid-quantity: go-testing-invalid-quantity-...
SAMPLE_OK testing
```

The valid orders return `PAY-2000` / `TRACK-ALICE-1` and `PAY-17499` /
`TRACK-BOB-2`. Invalid orders must be **Failed**, with the expected validation
cause. Failed instances are intentional and remain visible in the dashboard.
The activity bodies are illustrative business operations, not real payment or
shipping integrations.
