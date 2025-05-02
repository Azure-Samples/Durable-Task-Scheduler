# Function Chaining Pattern

## Description of the Sample

This sample demonstrates the function chaining pattern with the Azure Durable Task Scheduler using the Python SDK. Function chaining is a fundamental workflow pattern where activities are executed in a sequence, with the output of one activity passed as the input to the next activity.

In this sample:
1. The orchestrator calls the `say_hello` activity with a name input
2. The result is passed to the `process_greeting` activity
3. That result is passed to the `finalize_response` activity
4. The final result is returned to the client

This pattern is useful for:
- Creating sequential workflows where steps must execute in order
- Passing data between steps with data transformations at each step
- Building pipelines where each activity adds value to the result

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

There are two ways to run this sample: locally or deployed to Azure.

### Running Locally

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
python client.py [name]
```
You can optionally provide a name as an argument. If not provided, "User" will be used.

### Deploying with Azure Developer CLI (AZD)

This sample includes an `azure.yaml` configuration file that allows you to deploy the entire solution to Azure using Azure Developer CLI (AZD).

#### Prerequisites for AZD Deployment

1. Install [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
2. Authenticate with Azure:
   ```bash
   azd auth login
   ```

#### Deployment Steps

1. Navigate to the Function Chaining sample directory:
   ```bash
   cd /path/to/Durable-Task-Scheduler/samples/durable-task-sdks/python/function-chaining
   ```

2. Initialize the Azure Developer CLI project (only needed the first time):
   ```bash
   azd init
   ```
   This step prepares the environment for deployment and creates necessary configuration files.

3. Provision resources and deploy the application:
   ```bash
   azd up
   ```
   This command will:
   - Provision Azure resources (including Azure Container Apps and Durable Task Scheduler)
   - Build and deploy both the Client and Worker components
   - Set up the necessary connections between components

3. After deployment completes, AZD will display URLs for your deployed services.

4. Monitor your orchestrations using the Azure Portal by navigating to your Durable Task Scheduler resource.

5. To confirm the sample is working correctly, view the application logs through the Azure Portal:
   - Navigate to the Azure Portal (https://portal.azure.com)
   - Go to your resource group where the application was deployed
   - Find and select the Container Apps for both the worker and client components
   - For each Container App:
     - Click on "Log stream" in the left navigation menu under "Monitoring"
     - View the real-time logs showing orchestrations being scheduled, activities executing, and results being processed
   
   These logs will show the same information as when running locally, allowing you to confirm the application is working correctly.

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrator and activities
- Log entries when each activity is called, showing the input received at each step
- The progression through the chain of activities

### Client Output
The client shows:
- Starting the orchestration with the provided name
- The unique orchestration instance ID
- The final result, which should be a greeting composed from all three activities:
  - First activity: `Hello [name]!` 
  - Second activity: `Hello [name]! How are you today?`
  - Third activity: `Hello [name]! How are you today? I hope you're doing well!`

This demonstrates the chaining of functions in a sequence, with each function building on the result of the previous one.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The sequential execution of the three activities
   - The input and output at each step
   - The time taken for each step

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard visualizes the sequential nature of function chaining, making it easy to see the flow of data from one activity to the next.
