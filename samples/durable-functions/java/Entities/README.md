# Durable Entities — Durable Functions Java Sample

Java | Durable Functions

## Description

This sample demonstrates Durable Entities with Java using the Durable Task Scheduler backend. It includes:

1. A `Counter` durable entity with `add`, `subtract`, `get`, and `reset` operations
2. An HTTP endpoint for direct entity signals
3. An HTTP endpoint for direct entity reads
4. An orchestration that signals the entity and reads its final value

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
   cd samples/durable-functions/java/Entities
   mvn clean package
   ```

4. Run the Functions host:
   ```bash
   mvn azure-functions:run
   ```

5. Signal the entity directly:
   ```bash
   curl -X POST "http://localhost:7071/api/SignalCounter?key=my-counter&op=add&value=7"
   ```

6. Read the entity state:
   ```bash
   curl "http://localhost:7071/api/GetCounter?key=my-counter"
   ```

7. Start the orchestration:
   ```bash
   curl -X POST "http://localhost:7071/api/StartCounterOrchestration?key=my-orch-counter"
   ```

8. View orchestration activity in the dashboard: http://localhost:8082

## Notes

- `mvn clean package` is configured to stage the Azure Functions app so `mvn azure-functions:run` works as a separate second step.
- `AzureWebJobsStorage=UseDevelopmentStorage=true` requires Azurite to be running locally.
- `SignalCounter` rejects `op=get`; use `GetCounter` to read entity state.