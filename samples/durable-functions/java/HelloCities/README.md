# Hello Cities — Durable Functions Java Quickstart

Java | Durable Functions

## Description

This quickstart demonstrates Durable Functions with Java using the Durable Task Scheduler backend. It includes two patterns:

1. **Function Chaining** — An orchestration that calls three "say hello" activities sequentially for different cities
2. **Fan-out/Fan-in** — An orchestration that greets multiple cities in parallel and aggregates the results

## Prerequisites

1. [Java 11+](https://adoptium.net/) (JDK)
2. [Maven](https://maven.apache.org/download.cgi)
3. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
4. [Docker](https://www.docker.com/products/docker-desktop/) (for the DTS emulator and Azurite)

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Start Azurite for Azure Functions host storage:
   ```bash
   docker run --name azurite -d -p 10000:10000 -p 10001:10001 -p 10002:10002 mcr.microsoft.com/azure-storage/azurite
   ```

3. Build the sample:
   ```bash
   cd samples/durable-functions/java/HelloCities
   mvn clean package
   ```

4. Run the Functions host:
   ```bash
   mvn azure-functions:run
   ```

5. Trigger the function chaining orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartChaining
   ```

6. Trigger the fan-out/fan-in orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartFanOutFanIn
   ```

7. View in the dashboard: http://localhost:8082

## Notes

- `mvn clean package` is configured to stage the Azure Functions app so `mvn azure-functions:run` works as a separate second step.
- `AzureWebJobsStorage=UseDevelopmentStorage=true` requires Azurite to be running locally.
- `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` in `local.settings.json` points to the local DTS emulator on `http://localhost:8080`.

## Expected Output

The chaining orchestration greets Tokyo, Seattle, and London sequentially:
```
Hello Tokyo! Hello Seattle! Hello London!
```

The fan-out/fan-in orchestration greets all three cities in parallel and returns aggregated results.

## Learn More

- [Durable Functions Java API Reference](https://learn.microsoft.com/java/api/com.microsoft.durabletask.azurefunctions)
- [Durable Functions Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
