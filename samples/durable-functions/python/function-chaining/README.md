# Function Chaining Pattern

## Description of the Sample

This sample demonstrates the function chaining pattern with Azure Durable Functions in Python. Function chaining is a fundamental workflow pattern where activities are executed in a sequence, with the output of one activity passed as the input to the next activity.

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

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure Functions Core Tools](https://docs.microsoft.com/en-us/azure/azure-functions/functions-run-local) v4.x
3. [Docker](https://www.docker.com/products/docker-desktop/) (for running the Durable Task Scheduler) installed

## Configuring Durable Task Scheduler

There are two ways to run this sample locally:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Pull the Docker Image for the Emulator:
  ```bash
  docker pull mcr.microsoft.com/dts/dts-emulator:latest
  ```

1. Run the Emulator:
  ```bash
  docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
  ```
Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: http://localhost:8080, taskhub: default). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

Local development with a deployed scheduler:

1. Install the durable task scheduler CLI extension:

    ```bash
    az upgrade
    az extension add --name durabletask --allow-preview true
    ```

1. Create a resource group in a region where the Durable Task Scheduler is available:

    ```bash
    az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
    ```

    ```bash
    az group create --name my-resource-group --location <location>
    ```
1. Create a durable task scheduler resource:

    ```bash
    az durabletask scheduler create \
        --resource-group my-resource-group \
        --name my-scheduler \
        --ip-allowlist '["0.0.0.0/0"]' \
        --sku-name "Dedicated" \
        --sku-capacity 1 \
        --tags "{'myattribute':'myvalue'}"
    ```

1. Create a task hub within the scheduler resource:

    ```bash
    az durabletask taskhub create \
        --resource-group my-resource-group \
        --scheduler-name my-scheduler \
        --name "my-taskhub"
    ```

1. Grant the current user permission to connect to the `my-taskhub` task hub:

    ```bash
    subscriptionId=$(az account show --query "id" -o tsv)
    loggedInUser=$(az account show --query "user.name" -o tsv)

    az role assignment create \
        --assignee $loggedInUser \
        --role "Durable Task Data Contributor" \
        --scope "/subscriptions/$subscriptionId/resourceGroups/my-resource-group/providers/Microsoft.DurableTask/schedulers/my-scheduler/taskHubs/my-taskhub"
    ```

## How to Run the Sample

Once you have set up the Durable Task Scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

2. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

3. Start the Azure Functions runtime:
   ```bash
   func start
   ```
   
   You should see output indicating the functions have loaded successfully.

4. Test the orchestration by sending a POST request:
   ```bash
   # Start orchestration with default input
   curl -X POST http://localhost:7071/api/function_chaining \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start orchestration with custom name
   curl -X POST http://localhost:7071/api/function_chaining \
     -H "Content-Type: application/json" \
     -d '{"name": "Alice"}'
   ```

5. Check orchestration status using the `statusQueryGetUri` from the response:
   ```bash
   curl http://localhost:7071/api/status/{instanceId}
   ```

## Understanding the Output

When you run the sample, you'll see the following behavior:

1. **Initial Response**: The HTTP trigger returns a JSON response with management URLs:
   ```json
   {
     "id": "abcd1234",
     "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234",
     "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/raiseEvent/{eventName}",
     "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/terminate?reason={text}",
     "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234"
   }
   ```

2. **Orchestration Execution**: The function chaining pattern processes the input through three activities:
   - `say_hello`: Creates the initial greeting
   - `process_greeting`: Adds the "How are you today?" part
   - `finalize_response`: Adds the final "I hope you're doing well!" part

3. **Final Result**: For input `{"name": "Alice"}`, the orchestration will produce:
   ```
   "Hello Alice! How are you today? I hope you're doing well!"
   ```

## Dashboard Review

You can monitor the orchestration execution through the Durable Task Scheduler dashboard:

1. Navigate to `http://localhost:8082` in your browser
2. You'll see a list of task hubs - select the "default" hub
3. Click on your orchestration instance to see:
   - Orchestration timeline and execution steps
   - Input and output data for each activity
   - Performance metrics and timing information
   - Visual representation of the function chaining pattern

The dashboard provides real-time visibility into the orchestration execution, making it easy to understand how data flows through each step in the chain.

## Learn More

- [Function Chaining Pattern in Durable Functions](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-sequence)
- [Durable Task Scheduler Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-python)