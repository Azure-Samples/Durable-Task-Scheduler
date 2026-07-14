# Custom Exception Property Provider with Durable Functions (.NET)

.NET | Durable Functions

## Description

Demonstrates attaching **custom properties to `FailureDetails`** when an activity throws, using the Durable Task Scheduler (DTS) backend.

When an activity throws a `BusinessValidationException`, the registered `IExceptionPropertiesProvider` extracts structured fields (string, int, long, date-time, dictionary, list, null) from the exception. The Durable worker propagates those values onto `FailureDetails.Properties`, which the orchestration observes when it catches the failure — making rich, typed error context available to the orchestrator instead of just a message and stack trace.

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)

> **Note:** The custom exception property provider requires these minimum versions (the sample references the latest available):
> - `Microsoft.Azure.Functions.Worker.Extensions.DurableTask` >= 1.9.0
> - `Microsoft.DurableTask.Worker` >= 1.16.1 (provides `IExceptionPropertiesProvider`)
> - `Microsoft.Azure.Functions.Worker.Extensions.DurableTask.AzureManaged` >= 1.2.0 (for the Durable Task Scheduler backend)

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Start the Function app:
   ```bash
   cd samples/durable-functions/dotnet/CustomExceptionPropertyProvider
   func start
   ```

3. Start the orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/orchestrators/customException
   ```

4. Follow the `statusQueryGetUri` from the response to see the completed output.

5. View in the dashboard: http://localhost:8082

## Expected Output

The orchestration returns the caught `FailureDetails`, whose `Properties` carry the custom values supplied by the provider:

```json
{
  "errorType": "CustomExceptionPropertyProvider.BusinessValidationException",
  "errorMessage": "Business logic validation failed",
  "properties": {
    "StringProperty": "validation-error-123",
    "IntProperty": 100,
    "LongProperty": 999999999,
    "DateTimeProperty": "2025-10-15T14:30:00Z",
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

- `BusinessExceptionPropertiesProvider` implements `Microsoft.DurableTask.Worker.IExceptionPropertiesProvider` and is registered as a singleton in `Program.cs`.
- `BusinessActivity` throws a `BusinessValidationException` carrying the structured fields.
- `OrchestrationWithCustomException` calls the activity, catches the propagated `TaskFailedException`, and returns `ex.FailureDetails` — including the custom `Properties`.

## Learn More

- [Durable Functions error handling](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-error-handling)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
