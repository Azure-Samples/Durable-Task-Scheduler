<<<<<<< HEAD
# Monitoring Pattern with Azure Durable Task Scheduler

This sample demonstrates the monitoring pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern enables periodic checking of an external system or process until a certain condition is met or a timeout occurs.
=======
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
<<<<<<< HEAD
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the monitoring pattern by:

1. Periodically checking the status of a simulated external job
2. Updating the orchestration state with the latest status
3. Either completing when the job is done or timing out after a specified period

This pattern is useful for scenarios where you need to track the progress of an external system or process without blocking resources with a continuous connection.

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
<<<<<<< HEAD
=======
3. [Durable Task Scheduler resource](https://learn.microsoft.com/azure/durable-functions/durable-task-scheduler)
4. Appropriate Azure role assignments (Owner or Contributor)

## Setup

1. Create a virtual environment and activate it:

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
<<<<<<< HEAD
<<<<<<< HEAD
=======

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
```bash
pip install -r requirements.txt
```

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
3. Start the worker in a terminal:
```bash
python worker.py
```
You should see output indicating the worker has started and registered the orchestration and activities.

4. In a new terminal (with the virtual environment activated), run the client:
<<<<<<< HEAD
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

2. In a new terminal (with the virtual environment activated), run the client to start the monitoring orchestration:

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
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

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
### What Happens When You Run the Sample

When you run the sample:

1. The client creates an orchestration instance to monitor a job with the provided parameters.

2. The worker executes the `monitor_job` orchestration function, which:
   - Sets up initial monitoring parameters (job ID, polling interval, deadline)
   - Enters a loop that periodically checks the job status
   - Each iteration calls the `check_job_status` activity to simulate checking an external system
   - If the job completes or the deadline is reached, the orchestration completes
   - Otherwise, it schedules itself to wake up after the polling interval using `context.create_timer()`

3. The `check_job_status` activity simulates an external job that takes time to complete, with the completion percentage increasing on each check.

4. The client displays the status updates as the orchestration monitors the job until completion or timeout.

This sample demonstrates a pattern for monitoring long-running processes without maintaining a continuous connection, which is useful for tracking asynchronous operations in external systems.

<<<<<<< HEAD
## Viewing Orchestration Details in the Durable Task Dashboard

After running the sample, you can use the Durable Task Dashboard to view details about your monitoring orchestration:

1. Access the dashboard using the appropriate URL: `https://dashboard.durabletask.io`

2. When using an Azure deployed scheduler, you'll need to authenticate with an account that has been granted the "Durable Task Data Contributor" role.

3. In the dashboard, you'll see a list of all orchestration instances. Find your monitoring orchestration and click on it to see details.

4. The dashboard provides several views that are particularly useful for monitoring patterns:
   - **Timeline View**: Shows the recurring pattern of checks over time
   - **History View**: Details each monitoring iteration, including timestamps and status
   - **Sequence View**: Visualizes the workflow including continuous monitoring loops

The dashboard is particularly valuable for monitoring scenarios as it provides visibility into recurring orchestrations and helps identify trends or issues over extended periods of time.
=======
The orchestration will periodically check the job status until it completes or times out.

=======
>>>>>>> e8c58d1 (Continue improving READMEs)
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)
