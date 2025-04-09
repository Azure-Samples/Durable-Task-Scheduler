# Function Chaining Pattern

This sample demonstrates the function chaining pattern with the Azure Durable Task Scheduler using the Python SDK. In this pattern, an orchestration executes a sequence of functions in a specific order, where the output of one function becomes the input to the next function.

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
python client.py "YourName"
```

The client will schedule a new orchestration instance and wait for it to complete. The worker will execute the orchestration, which chains three activities together in sequence:

1. The first activity creates a greeting with your name
2. The second activity processes that greeting
3. The third activity finalizes the response

Each activity's output serves as the input to the next activity. The final result is returned to the client.

## Sample Explanation

The function chaining pattern is useful for workflows where steps must be executed in a specific order, and each step depends on the output of the previous step. Examples include:

- Processing pipelines
- Approval workflows with multiple steps
- Data transformation chains

In this sample, the pattern is demonstrated through a simple series of greeting transformations, where each activity builds upon the output of the previous activity.
