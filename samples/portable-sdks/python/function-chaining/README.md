# Function Chaining Pattern

This sample demonstrates the function chaining pattern with the Azure Durable Task Scheduler using the Python SDK. In this pattern, an orchestration executes a sequence of functions in a specific order, where the output of one function becomes the input to the next function.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration chains three activities together in sequence:

1. The first activity creates a greeting with your name
2. The second activity processes that greeting
3. The third activity finalizes the response

Each activity's output serves as the input to the next activity. The final result is returned to the client.

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

## Running the Sample

1. First, start the worker that registers the activities and orchestrations:

```bash
python worker.py
```

2. In a new terminal (with the virtual environment activated), run the client to start the orchestration:

```bash
python client.py "YourName"
```

The client will schedule a new orchestration instance and wait for it to complete.

## Sample Explanation

The function chaining pattern is useful for workflows where steps must be executed in a specific order, and each step depends on the output of the previous step. Examples include:

- Processing pipelines
- Approval workflows with multiple steps
- Data transformation chains

In this sample, the pattern is demonstrated through a simple series of greeting transformations, where each activity builds upon the output of the previous activity.
