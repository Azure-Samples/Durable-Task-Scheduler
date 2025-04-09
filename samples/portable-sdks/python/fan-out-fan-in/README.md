# Fan Out/Fan In Pattern

This sample demonstrates the fan out/fan in pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern allows you to execute multiple tasks in parallel and then aggregate their results.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the fan out/fan in pattern by:

1. Spawning multiple parallel activity tasks (fan out)
2. Waiting for all activities to complete
3. Aggregating the results of all activities (fan in)

This pattern is useful for parallel processing scenarios where you need to combine results.

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
$env:TASKHUB="your-taskhub-name"
$env:ENDPOINT="your-scheduler-endpoint"
```

## Running the Sample

1. First, start the worker that registers the activities and orchestrations:

```bash
python worker.py
```

2. In a new terminal (with the virtual environment activated), run the client to start the orchestration:

```bash
python client.py 20  # Optionally specify the number of work items (default is 10)
```rates the fan out/fan in pattern with the Azure Durable Task Scheduler using the Python SDK. In this pattern, the orchestrator executes multiple functions in parallel and then waits for all of them to finish before aggregating the results.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Durable Task Scheduler resource](https://learn.microsoft.com/azure/durable-functions/durable-task-scheduler)
4. Appropriate Azure role assignments (Owner or Contributor)

## Setup

1. Create a virtual environment and activate it:

```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:

```bash
pip install -r requirements.txt
```

3. Make sure you're logged in to Azure:

```bash
az login
```

## Running the Sample

1. First, start the worker that registers the activities and orchestrations:

```bash
python worker.py
```

2. In a new terminal (with the virtual environment activated), run the client to start the orchestration:

```bash
python client.py 10
```

The parameter `10` specifies the number of work items to process in parallel. You can adjust this value as needed.

The client will schedule a new orchestration instance and wait for it to complete. The worker will:
1. Fan out: Process multiple work items in parallel 
2. Fan in: Wait for all parallel executions to complete
3. Aggregate the results of all work items

## Sample Explanation

The fan out/fan in pattern is ideal for workloads that can be parallelized into independent tasks, with results combined afterward. Common use cases include:

- Data processing (map-reduce pattern)
- Batch processing of independent items
- Parallel API calls or data retrieval
- High-performance computation

This pattern enables efficient processing by:
1. Distributing work across multiple parallel executions
2. Processing items concurrently to reduce total execution time
3. Combining the results into a single aggregated output

In this sample, each work item is processed independently in parallel, and the results are aggregated into summary statistics.
