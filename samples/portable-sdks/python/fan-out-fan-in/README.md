# Fan Out/Fan In Pattern

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
