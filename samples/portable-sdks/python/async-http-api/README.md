# Async HTTP API Pattern

This sample demonstrates the async HTTP API pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern allows you to start long-running operations via HTTP and retrieve their results once they complete, without forcing clients to wait synchronously.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the async HTTP API pattern by:

1. Starting a long-running operation asynchronously
2. Returning a status URL immediately to the client
3. Processing the request in the background
4. Making the result available for retrieval when complete

This pattern is ideal for implementing RESTful services with long-running operations, avoiding the need to keep HTTP connections open for extended periods.

## Running the Examples

There are two separate ways to run an example:

- Using the Emulator
- Using a deployed Scheduler and Taskhub

### Running with a Deployed Scheduler and Taskhub Resource

1. To create a taskhub, follow these steps using the Azure CLI commands:

Create a Scheduler:

```bash
az durabletask scheduler create --resource-group --name --location --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
```

Create Your Taskhub:

```bash
az durabletask taskhub create --resource-group <testrg> --scheduler-name <testscheduler> --name <testtaskhub>
```

2. Retrieve the Endpoint for the Scheduler: Locate the taskhub in the Azure portal to find the endpoint.

3. Set the Environment Variables:

Bash:
```bash
export TASKHUB=<taskhubname>
export ENDPOINT=<taskhubEndpoint>
```

Powershell:
```powershell
$env:TASKHUB = "<taskhubname>"
$env:ENDPOINT = "<taskhubEndpoint>"
```

4. Install the Correct Packages:
```bash
pip install -r requirements.txt
```

5. Grant your developer credentials the Durable Task Data Contributor Role.

### Running with the Emulator

The emulator simulates a scheduler and taskhub, packaged into an easy-to-use Docker container. For these steps, it is assumed that you are using port 8080.

1. Install Docker: If it is not already installed.

2. Pull the Docker Image for the Emulator:

```bash
docker pull mcr.microsoft.com/dts/dts-emulator:v0.0.6
```

3. Run the Emulator: Wait a few seconds for the container to be ready.

```bash
docker run --name dtsemulator -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:v0.0.4
```

4. Set the Environment Variables:

Bash:
```bash
export TASKHUB=<taskhubname>
export ENDPOINT=http://localhost:8080
```

Powershell:
```powershell
$env:TASKHUB = "<taskhubname>"
$env:ENDPOINT = "http://localhost:8080"
```

5. Edit the Examples: Change the `token_credential` input of both the `DurableTaskSchedulerWorker` and `DurableTaskSchedulerClient` to `None`.

## Sample Explanation

The async HTTP API pattern is useful for implementing RESTful services with long-running operations. Instead of keeping an HTTP connection open for the entire operation, this pattern:

1. Returns an immediate response with a status URL
2. Processes the request asynchronously in the background
3. Allows the client to check the status via the provided URL
4. Returns the final result when the operation completes

This pattern is common in many real-world scenarios:
- File processing services
- Data import/export operations
- Complex calculations or analysis
- Resource provisioning

In this sample, the FastAPI application demonstrates how to use durable tasks to manage the lifecycle of asynchronous operations while providing a responsive HTTP API.
