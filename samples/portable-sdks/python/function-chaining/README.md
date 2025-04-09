# Function Chaining Pattern

This sample demonstrates the function chaining pattern with the Azure Durable Task Scheduler using the Python SDK. In this pattern, an orchestration executes a sequence of functions in a specific order, where the output of one function becomes the input to the next function.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
<<<<<<< HEAD
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration chains three activities together in sequence:
=======
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)

1. The first activity creates a greeting with your name
2. The second activity processes that greeting
3. The third activity finalizes the response

Each activity's output serves as the input to the next activity. The final result is returned to the client.

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
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
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
```bash
pip install -r requirements.txt
```

3. Start the worker in a terminal:
```bash
python worker.py
```
You should see output indicating the worker has started and registered the orchestration and activities.

4. In a new terminal (with the virtual environment activated), run the client:
```bash
python client.py [your-name]
```

For example:
```bash
python client.py Alice
```
<<<<<<< HEAD
=======

If you don't provide a name, the script will use a default name.

### What Happens When You Run the Sample

When you run the sample:

1. The client creates an orchestration instance with your provided name as input.

2. The worker executes the `function_chain` orchestration function, which:
   - Calls the `create_greeting` activity with your name
   - Takes that greeting and passes it to the `process_greeting` activity
   - Takes the processed greeting and passes it to the `finalize_greeting` activity
   - Returns the final result to the client

3. Each activity in the chain:
   - `create_greeting`: Generates a simple greeting string with your name
   - `process_greeting`: Transforms the greeting by adding additional text
   - `finalize_greeting`: Formats the final result with additional styling

4. The client displays the final result from the completed orchestration.

This sample demonstrates how to create sequential workflows where the output of one step serves as the input to the next step. This pattern is useful for creating multi-step processes where each step depends on the result of the previous step.
>>>>>>> e8c58d1 (Continue improving READMEs)

If you don't provide a name, the script will use a default name.

### What Happens When You Run the Sample

When you run the sample:

1. The client creates an orchestration instance with your provided name as input.

2. The worker executes the `function_chain` orchestration function, which:
   - Calls the `create_greeting` activity with your name
   - Takes that greeting and passes it to the `process_greeting` activity
   - Takes the processed greeting and passes it to the `finalize_greeting` activity
   - Returns the final result to the client

3. Each activity in the chain:
   - `create_greeting`: Generates a simple greeting string with your name
   - `process_greeting`: Transforms the greeting by adding additional text
   - `finalize_greeting`: Formats the final result with additional styling

4. The client displays the final result from the completed orchestration.

This sample demonstrates how to create sequential workflows where the output of one step serves as the input to the next step. This pattern is useful for creating multi-step processes where each step depends on the result of the previous step.

## Viewing Orchestration Details in the Durable Task Dashboard

After running the sample, you can use the Durable Task Dashboard to view details about your function chaining orchestration:

1. Access the dashboard using the appropriate URL: `https://dashboard.durabletask.io`

2. When using an Azure deployed scheduler, you'll need to authenticate with an account that has been granted the "Durable Task Data Contributor" role.

3. In the dashboard, you'll see a list of all orchestration instances. Find your function chaining orchestration and click on it to see details.

4. The dashboard provides several views that are particularly useful for function chaining patterns:
   - **Sequence View**: Visualizes the sequential flow of functions in the chain
   - **History View**: Shows each function execution in the chain with timestamps and status
   - **Timeline View**: Illustrates the duration and sequence of each function in the chain

The dashboard provides valuable insights into how your chained functions execute, helping you optimize performance and identify issues.
=======
## Sample Explanation

The function chaining pattern is useful for workflows where steps must be executed in a specific order, and each step depends on the output of the previous step. Examples include:

- Processing pipelines
- Approval workflows with multiple steps
- Data transformation chains

In this sample, the pattern is demonstrated through a simple series of greeting transformations, where each activity builds upon the output of the previous activity.
<<<<<<< HEAD
>>>>>>> 8b26beb (Add python samples for the durable app patterns)
=======

Function chaining is a fundamental pattern for orchestrating sequential workflows where:

1. Operations must be performed in a specific order
2. Each operation depends on the result of the previous one
3. The entire sequence represents a single coherent workflow

Common use cases include:
- Multi-step data processing pipelines
- Document approval workflows
- Sequential validation processes
- ETL (Extract, Transform, Load) operations

In this sample, the orchestration chains three simple text processing activities together, with each one building upon the result of the previous activity to produce a final message.
>>>>>>> e8c58d1 (Continue improving READMEs)
