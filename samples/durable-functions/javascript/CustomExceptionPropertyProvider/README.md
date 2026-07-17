# Custom Exception Property Provider — Durable Functions JavaScript

JavaScript | Durable Functions

## Description

Demonstrates attaching **custom properties to `FailureDetails`** when an activity throws, using the Node.js v4 programming model and the Durable Task Scheduler (DTS) backend.

An activity throws a `BusinessValidationException` carrying structured fields (string, int, long, date-time, dictionary, list, null). A provider registered via `df.app.setExceptionPropertiesProvider(...)` extracts those fields, and the Durable worker propagates them onto `FailureDetails.Properties`. The orchestration catches the propagated `TaskFailedError` and returns its `failureDetails`, so the rich error context is available to the orchestrator instead of just a message and stack trace.

## Prerequisites

1. [Node.js 18+](https://nodejs.org/) (use Node 22 LTS; the Functions host does not yet support Node 24)
2. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

> **Note:** The custom exception property provider requires:
> - The `durable-functions` npm package >= **3.5.0**, which adds `df.app.setExceptionPropertiesProvider`.
> - An extension bundle that ships the feature. Use the **main** bundle `Microsoft.Azure.Functions.ExtensionBundle` >= **4.37.1** (configured in `host.json`), or the **preview** bundle `Microsoft.Azure.Functions.ExtensionBundle.Preview` >= **4.44.0**. Older bundles will **not** surface `Properties`.
> - The Durable Task Scheduler (azure-managed) SDK >= **1.2.0**.

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Install dependencies and run:
   ```bash
   cd samples/durable-functions/javascript/CustomExceptionPropertyProvider
   npm install
   func start
   ```

3. Start the orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartCustomException
   ```

4. Follow the `statusQueryGetUri` from the response to see the completed output.

5. View in the dashboard: http://localhost:8082

## Expected Output

The orchestration returns the caught `failureDetails`, whose `properties` carry the custom values supplied by the provider:

```json
{
  "errorType": "BusinessValidationException",
  "errorMessage": "Business logic validation failed",
  "properties": {
    "StringProperty": "validation-error-123",
    "IntProperty": 100,
    "LongProperty": 999999999,
    "DateTimeProperty": "2025-10-15T14:30:00.000Z",
    "DictionaryProperty": {
      "error_code": "VALIDATION_FAILED",
      "retry_count": 3,
      "is_critical": true
    },
    "ListProperty": ["error1", "error2", 500, null],
    "NullProperty": null
  }
}
```

## How It Works

- `df.app.setExceptionPropertiesProvider({ getExceptionProperties(error) { ... } })` registers a global provider once at app startup.
- `businessActivity` throws a `BusinessValidationException` carrying the structured fields.
- `orchestrationWithCustomException` calls the activity, catches the propagated `TaskFailedError`, and returns `e.failureDetails` — including the custom `properties`.

## Learn More

- [Durable Functions JavaScript API Reference](https://learn.microsoft.com/javascript/api/durable-functions/)
- [Durable Functions error handling](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-error-handling)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
