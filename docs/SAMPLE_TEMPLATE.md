# [Sample Name]

[Language] | [Framework: Durable Task SDK / Durable Functions]

## Description

[2-3 sentences explaining what this sample demonstrates, what pattern it uses, and why it's useful.]

## Prerequisites

1. [Language runtime] (e.g., .NET 8 SDK, Python 3.9+, Java 17+)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Any additional prerequisites]

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. [Install dependencies / build step]:
   ```bash
   [command]
   ```

3. Start the worker:
   ```bash
   [command]
   ```

4. In a new terminal, run the client:
   ```bash
   [command]
   ```

## Expected Output

[Show what the user should see when running the sample, e.g.:]

```
Started orchestration with ID: abc123
Waiting for completion...
Orchestration completed: [result]
```

## Using a Deployed Scheduler (Azure)

To use a Durable Task Scheduler in Azure instead of the emulator:

1. Set environment variables:
   ```bash
   export ENDPOINT=<your-scheduler-endpoint>
   export TASKHUB=<your-taskhub-name>
   ```

2. Run the sample using the same commands as above.

See the [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler) for setup instructions.

## Code Walkthrough

[Brief explanation of the key code, highlighting the pattern being demonstrated.]

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

## Related Samples

- [Link to related sample 1]
- [Link to related sample 2]

## Learn More

- [Relevant Microsoft Learn link]
- [Pattern documentation link]
