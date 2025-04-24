# Java Samples for Durable Task SDK

This directory contains sample applications demonstrating various patterns using the Durable Task Java SDK.

## Prerequisites

- Java 11 or later
- Gradle 7.4 or above
- Azure Durable Task Scheduler connection string (see [Getting Started with Durable Task Scheduler](https://learn.microsoft.com/en-us/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler?tabs=function-app-integrated-creation&pivots=az-cli) for setup instructions)

## Setup

1. Set the connection string environment variable:
   ```bash
   # Windows
   set DURABLE_TASK_CONNECTION_STRING=your_connection_string

   # Linux/macOS
   export DURABLE_TASK_CONNECTION_STRING=your_connection_string
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