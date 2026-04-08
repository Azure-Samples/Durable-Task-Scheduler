### Azure Durable Functions (Java)

[Durable Functions](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview) is an extension of Azure Functions that lets you write stateful functions in a serverless compute environment.

These samples demonstrate how to use Durable Functions with Java and the Durable Task Scheduler backend.

## Available Samples

- **HelloCities**: Function chaining and fan-out/fan-in patterns
- **Entities**: Durable entities with a counter example (add, subtract, get, reset operations)

## Local Run Model

Before running either Java sample locally, start the required services:

```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
docker run --name azurite -d -p 10000:10000 -p 10001:10001 -p 10002:10002 mcr.microsoft.com/azure-storage/azurite
```

Then run the sample using the same two Maven commands as the Java Durable Functions quickstart:

```bash
mvn clean package
mvn azure-functions:run
```

`mvn clean package` stages the Azure Functions app, and `mvn azure-functions:run` starts the local Functions host from that staged output.
