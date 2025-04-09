# Monitoring Pattern with Azure Durable Task Scheduler

This sample demonstrates the monitoring pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern enables periodic checking of an external system or process until a certain condition is met or a timeout occurs.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the monitoring pattern by:

1. Periodically checking the status of a simulated external job
2. Updating the orchestration state with the latest status
3. Either completing when the job is done or timing out after a specified period

This pattern is useful for scenarios where you need to track the progress of an external system or process without blocking resources with a continuous connection.

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

The monitoring pattern is useful for scenarios where you need to track the progress of an external process or system that may take a while to complete. Instead of blocking resources with a continuous connection, this pattern:

1. Checks the status of the external system periodically
2. Sleeps between checks to conserve resources
3. Completes when either the desired condition is met or a timeout occurs

Common use cases include:
- Monitoring asynchronous job status
- Waiting for resource provisioning to complete
- Polling for file creation or changes
- Checking for availability of services or data

In this sample, the orchestration simulates monitoring an external job by periodically checking its status until it completes successfully or reaches the specified timeout.
