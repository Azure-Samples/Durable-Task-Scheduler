# Fan Out/Fan In Pattern

<<<<<<< HEAD
This sample demonstrates the fan out/fan in pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern allows you to execute multiple tasks in parallel and then aggregate their results.
=======
3. Make sure you're logged in to Azure:

```bash
az login
```

4. Set up the required environment variables:

```bash
# For bash/zsh
export TASKHUB="your-taskhub-name"
export ENDPOINT="your-scheduler-endpoint"

# For Windows PowerShell
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
<<<<<<< HEAD
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the fan out/fan in pattern by:

1. Spawning multiple parallel activity tasks (fan out)
2. Waiting for all activities to complete
3. Aggregating the results of all activities (fan in)

This pattern is useful for parallel processing scenarios where you need to combine results.

## Configuring the Sample

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

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment:
=======
3. [Durable Task Scheduler resource](https://learn.microsoft.com/azure/durable-functions/durable-task-scheduler)
4. Appropriate Azure role assignments (Owner or Contributor)

## Setup

1. Create a virtual environment and activate it:

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
<<<<<<< HEAD
=======

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
```bash
pip install -r requirements.txt
```

<<<<<<< HEAD
3. Start the worker in a terminal:
```bash
python worker.py
```
You should see output indicating the worker has started and registered the orchestration and activities.

4. In a new terminal (with the virtual environment activated), run the client:
```bash
python client.py
```

### What Happens When You Run the Sample

When you run the sample:

1. The client creates an instance of `DurableTaskClient` and starts a new orchestration instance.

2. The worker executes the `fan_out_fan_in` orchestration function, which:
   - Generates a list of work items to process
   - Creates tasks for each work item and executes them in parallel (fan out phase)
   - Uses `Task.all_settled()` to wait for all parallel tasks to complete
   - Aggregates the results from all completed activities (fan in phase)
   - Returns the final aggregated result

3. The client waits for the orchestration to complete and displays the final combined result.

### Viewing Orchestration Details in the Durable Task Dashboard

After running the sample, you can use the Durable Task Dashboard to view details about your orchestration execution:

1. Access the dashboard using the appropriate URL: `https://dashboard.durabletask.io`

2. When using an Azure deployed scheduler, you'll need to authenticate with an account that has been granted the "Durable Task Data Contributor" role.

3. In the dashboard, you'll see a list of all orchestration instances. Find your fan-out-fan-in orchestration and click on it to see details.

4. The dashboard provides several views of your orchestration:
   - **Timeline View**: Shows the parallel execution of your multiple activities, making it easy to visualize how tasks run concurrently
   - **History View**: Provides detailed event sequence with timestamps for all parallel activities
   - **Sequence View**: Visualizes the event sequence, clearly showing the fan-out and fan-in phases
   - **Input/Output View**: Shows the inputs and outputs of each parallel activity

5. You can also see the status (Running, Completed, Failed) and manage orchestrations using the dashboard controls.

The dashboard is particularly useful for fan-out-fan-in patterns as it visually shows how activities run in parallel and how their results are aggregated, helping you identify performance bottlenecks or failures in any of the parallel branches.
=======
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)
