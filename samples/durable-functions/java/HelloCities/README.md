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
4. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

## Quick Run

1. Start the emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Build and run:
   ```bash
   cd samples/durable-functions/java/HelloCities
   mvn clean package
   func start
   ```

3. Trigger the function chaining orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartChaining
   ```

4. Trigger the fan-out/fan-in orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartFanOutFanIn
   ```

5. View in the dashboard: http://localhost:8082

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
