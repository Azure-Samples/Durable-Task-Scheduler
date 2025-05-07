# Java Samples for Durable Task SDK

This directory contains sample applications demonstrating various patterns using the Durable Task Java SDK.

## Prerequisites

- Java 8 or later
- [Docker](https://www.docker.com/get-started)

## Running the samples with the Durable Task Scheduler Emulator

The samples can be run locally using the Durable Task Scheduler Emulator. The emulator is a containerized version of the Durable Task Scheduler service that persists state in memory. It is useful for development and testing purposes.

1. Start the Durable Task Scheduler Emulator:

   ```bash
   # Pull the emulator image
   docker pull mcr.microsoft.com/dts/dts-emulator:latest

   # Run the emulator
   docker run -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Set the connection string environment variable:
   ```bash
   # Windows
   set DURABLE_TASK_CONNECTION_STRING=Endpoint=http://localhost:8080;TaskHub=default;Authentication=None

   # Linux/macOS
   export DURABLE_TASK_CONNECTION_STRING=Endpoint=http://localhost:8080;TaskHub=default;Authentication=None
   ```

## Available Samples

Each sample demonstrates a different orchestration pattern:

- **async-http-api**: Basic HTTP API sample with rest endpoints to schedule/query orchestration instances.
- **function-chaining**: Sequential execution of multiple functions in a specific order
- **fan-out-fan-in**: Parallel execution of multiple functions and aggregating their results
- **eternal-orchestrations**: Long-running orchestrations that process work items periodically
- **human-interaction**: Integration of human approval steps in orchestrations
- **monitoring**: Monitoring and tracking orchestration progress
- **sub-orchestrations**: Composing multiple orchestrations hierarchically

## Running the Samples

Navigate to the specific sample directory and use Gradle to run the sample:

```bash
# For async-http-api sample
cd async-http-api
./gradlew runWebApi

# For function-chaining sample
cd function-chaining
./gradlew runChainingPattern

# For fan-out-fan-in sample
cd fan-out-fan-in
./gradlew runFanOutFanInPattern

# For eternal-orchestrations sample
cd eternal-orchestrations
./gradlew runEternalOrchestration

# For human-interaction sample
cd human-interaction
./gradlew runHumanInteraction

# For monitoring sample
cd monitoring
./gradlew runMonitoringPattern

# For sub-orchestrations sample
cd sub-orchestrations
./gradlew runSubOrchestrationPattern
```

## View orchestrations in the dashboard

You can view the orchestrations in the Durable Task Scheduler emulator's dashboard by navigating to `http://localhost:8082` in your browser and selecting the `default` task hub.

## Using Azure Durable Task Scheduler

To use the Azure Durable Task Scheduler instead of the emulator, set the connection string to your Durable Task Scheduler connection string:

```bash
# Windows
set DURABLE_TASK_CONNECTION_STRING=your_azure_connection_string

# Linux/macOS
export DURABLE_TASK_CONNECTION_STRING=your_azure_connection_string
```
