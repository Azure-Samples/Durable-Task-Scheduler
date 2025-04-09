# Monitoring Pattern

This3. Make sure you're logged in to Azure:

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
python client.py
```s the monitoring pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern enables periodic checking of an external system or process until a certain condition is met or a timeout occurs.

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

2. In a new terminal (with the virtual environment activated), run the client to start the monitoring orchestration:

```bash
python client.py [job_id] [polling_interval] [timeout]
```

For example:
```bash
python client.py job-123 5 30
```

Where:
- `job_id` is an optional identifier for the job (defaults to a generated UUID)
- `polling_interval` is the number of seconds between status checks (defaults to 5)
- `timeout` is the maximum number of seconds to monitor before timing out (defaults to 30)

The orchestration will periodically check the job status until it completes or times out.

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
