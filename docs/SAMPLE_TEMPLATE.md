# [Sample Name]

[Language] | [Framework: Durable Task SDK / Durable Functions]

## Description

[2-3 sentences explaining what this sample demonstrates, what pattern it uses, and why it's useful.]

## Prerequisites

1. [Language runtime] (e.g., .NET 8 SDK, Python 3.9+, Java 17+, Go 1.25.0+)
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

3. Start the worker (or the combined worker/client if the sample runs both):
   ```bash
   [command]
   ```

4. If the sample has a separate client, run it in a new terminal:
   ```bash
   [command]
   ```

For Go SDK samples, use the shared module at `samples/durable-task-sdks/go`: run `go mod download`, then `go run ./<sample-name>`. Do not create a nested module. The sample should run its worker and client together, verify its results, and exit. Go is not a Durable Functions language.

## Expected Output

[Show what the user should see when running the sample, e.g.:]

```
Started orchestration with ID: abc123
Waiting for completion...
Orchestration completed: [result]
```

## Using a Deployed Scheduler (Azure)

To use a Durable Task Scheduler in Azure instead of the emulator:

1. Set the environment variables that the sample actually reads. For samples using separate endpoint and task-hub variables:
   ```bash
   export ENDPOINT=<your-scheduler-endpoint>
   export TASKHUB=<your-taskhub-name>
   ```

   For Go SDK samples, use a connection string instead:
   ```bash
   export DTS_CONNECTION_STRING='Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure'
   ```
   Authenticate with `az login` for local development and grant the identity the Durable Task Data Contributor role. Use `Authentication=AzureCLI` to explicitly select CLI authentication. See the [Go Azure connection instructions](./quickstart.md#connect-the-go-samples-to-azure).

2. Run the sample using the same commands as above.

Document resource provisioning separately from connecting to an existing scheduler. Only advertise deployment templates or `azd up` when that sample actually includes them.

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
