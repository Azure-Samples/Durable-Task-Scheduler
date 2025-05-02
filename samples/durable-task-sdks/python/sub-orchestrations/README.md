# Sub-Orchestrations Pattern

## Description of the Sample

This sample demonstrates the sub-orchestrations pattern with the Azure Durable Task Scheduler using the Python SDK. Sub-orchestrations allow you to break down complex workflows into smaller, reusable components that can be called from parent orchestrations.

In this sample:
1. The main orchestrator function gets a list of orders from an activity function
2. For each order, it starts a sub-orchestration to process it
3. Each sub-orchestration runs a sequence of activities (inventory check, payment, shipping, notification)
4. The main orchestrator aggregates the results from all sub-orchestrations

Sub-orchestrations are useful for:
- Breaking down complex workflows into simpler, more maintainable pieces
- Reusing common workflow patterns across different orchestrations
- Processing collections of items in parallel

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Install Docker if it's not already installed.

2. Pull the Docker Image for the Emulator:
```bash
docker pull mcr.microsoft.com/dts/dts-emulator:v0.0.6
```

3. Run the Emulator:
```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:v0.0.6
```
Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: http://localhost:8080, taskhub: default). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

For production scenarios or when you're ready to deploy to Azure:

1. Create a Scheduler using the Azure CLI:
```bash
az durabletask scheduler create --resource-group <testrg> --name <testscheduler> --location <eastus> --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
```

2. Create Your Taskhub:
```bash
az durabletask taskhub create --resource-group <testrg> --scheduler-name <testscheduler> --name <testtaskhub>
```

3. Retrieve the Endpoint for the Scheduler from the Azure portal.

4. Set the Environment Variables:

   Bash:
   ```bash
   export TASKHUB=<taskhubname>
   export ENDPOINT=<taskhubEndpoint>
   ```

   PowerShell:
   ```powershell
   $env:TASKHUB = "<taskhubname>"
   $env:ENDPOINT = "<taskhubEndpoint>"
   ```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
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

4. In a new terminal (with the virtual environment activated if applicable), run the client:
```bash
python client.py
```

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrators and activities
- Notification when orders are generated
- Status updates as each order is processed through the various steps (inventory check, payment, shipping, notification)
- Each step has a random chance of succeeding or failing

### Client Output
The client shows:
- Starting the orchestration
- The final result, which includes:
  - The list of orders that were processed
  - How many orders were completed successfully
  - How many orders failed
  - Detailed results for each order, including failure reasons if applicable

The example demonstrates how multiple sub-orchestrations can run in parallel, with each managing its own workflow of activities.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to see:
   - The main orchestration
   - All the sub-orchestrations it created
   - The activities that were called
   - The inputs and outputs at each step

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard helps visualize the parent-child relationship between the main orchestration and its sub-orchestrations, making it easier to understand the flow and identify any issues.
