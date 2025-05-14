# Function Chaining Pattern

## Description of the Sample

This sample demonstrates the function chaining pattern with the Azure Durable Task Scheduler using the Java SDK. Function chaining is a fundamental workflow pattern where activities are executed in a sequence, with the output of one activity passed as the input to the next activity.

In this sample:
1. Receive input
The orchestrator gets a string input (e.g., `"Hello, world!"`).
2. Call `Reverse` activity
Reverses the input string.  
Example: `"Hello, world!"` → `"!dlrow ,olleH"`.
3. Call `Capitalize` activity
Converts the reversed string to uppercase.  
Example: `"!dlrow ,olleH"` → `"!DLROW ,OLLEH"`.
4. Call `ReplaceWhitespace` activity
Replaces whitespace characters with dashes (`-`).  
Example: `"!DLROW ,OLLEH"` → `"!DLROW-,OLLEH"`.
5. Complete orchestration
Returns the final transformed string as the orchestration output.

This pattern is useful for:
- Creating sequential workflows where steps must execute in order
- Passing data between steps with data transformations at each step
- Building pipelines where each activity adds value to the result

## Prerequisites

- Java 8 or later
- [Docker](https://www.docker.com/get-started)
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Install Docker if it's not already installed.

2. Pull the Docker Image for the Emulator:
```bash
docker pull mcr.microsoft.com/dts/dts-emulator:latest
```

3. Run the Emulator:
```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
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

4. Set the connection string environment variable:
   ```bash
   # Windows
   set DURABLE_TASK_CONNECTION_STRING=Endpoint=http://localhost:8080;TaskHub=default;Authentication=None

   # Linux/macOS
   export DURABLE_TASK_CONNECTION_STRING="Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
   ```

## How to Run the Sample

There are two ways to run this sample: locally or deployed to Azure.

### Running Locally

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

cd function-chaining
./gradlew runChainingPattern

### Deploying with Azure Developer CLI (AZD)

This sample includes an `azure.yaml` configuration file that allows you to deploy the sample to Azure using Azure Developer CLI (AZD).

#### Prerequisites for AZD Deployment

1. Install [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
2. Authenticate with Azure:
   ```bash
   azd auth login
   ```

#### Deployment Steps

1. Navigate to the Function Chaining sample directory:
   ```bash
   cd /path/to/Durable-Task-Scheduler/samples/durable-task-sdks/java/function-chaining
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
   - Build and deploy components
   - Set up the necessary connections between components

3. After deployment completes, AZD will display URLs for your deployed services.

4. Monitor your orchestrations using the Azure Portal by navigating to your Durable Task Scheduler resource.

5. To confirm the sample is working correctly, view the application logs through the Azure Portal:
   - Navigate to the Azure Portal (https://portal.azure.com)
   - Go to your resource group where the application was deployed
   - Find and select the Container App
   - Click on "Log stream" in the left navigation menu under "Monitoring"
   - View the real-time logs showing orchestrations being scheduled, activities executing, and results being processed

   These logs will show the same information as when running locally, allowing you to confirm the application is working correctly.

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

The worker shows:
- Registration of the orchestrator and activities
- Log entries when each activity is called, showing the input received at each step
- The progression through the chain of activities

The client shows:
- Starting the orchestration with the provided name
- The unique orchestration instance ID
- The final result, which should be a greeting composed from all three activities:
  - First activity: `"Hello, world!"` → `"!dlrow ,olleH"` (Reverse)
  - Second activity: `"!dlrow ,olleH"` → `"!DLROW ,OLLEH"` (Capitalize)
  - Third activity: `"!DLROW ,OLLEH"` → `"!DLROW-,OLLEH"` (ReplaceWhitespace)

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
