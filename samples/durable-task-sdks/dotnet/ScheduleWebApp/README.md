# Schedule Orchestrations with Durable Task SDK for .NET

This sample demonstrates a web application that leverages the durable task scheduler to manage recurring background tasks. Users can create, update, pause, resume, and delete schedules through REST endpoints provided by the ScheduleController. A dedicated orchestrator (CacheClearingOrchestrator) is used to perform periodic operations, such as clearing caches, by automatically triggering the scheduled tasks at configured intervals.

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample locally: locally: 

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Pull the Docker Image for the Emulator:
    ```bash
    docker pull mcr.microsoft.com/dts/dts-emulator:latest
    ```

1. Run the Emulator:
    ```bash
    docker run -it -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
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

## Authentication

The sample includes smart detection of the environment and configures authentication automatically:

- For local development with the emulator (when endpoint is http://localhost:8080), no authentication is required.
- For local development with a deployed scheduler, DefaultAzure authentication is used, which utilizes DefaultAzureCredential behind the scenes and tries multiple authentication methods:
  - Managed Identity
  - Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
  - Azure CLI login
  - Visual Studio login
  - and more

The connection string is constructed dynamically based on the environment:
```csharp
// For local emulator
connectionString = $"Endpoint={schedulerEndpoint};TaskHub={taskHubName};Authentication=None";

// For Azure deployed emulator
connectionString = $"Endpoint={schedulerEndpoint};TaskHub={taskHubName};Authentication=DefaultAzure";
```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1.  If you're using a deployed scheduler, you need to set Environment Variables.
    ```bash
    export ENDPOINT=$(az durabletask scheduler show \
        --resource-group my-resource-group \
        --name my-scheduler \
        --query "properties.endpoint" \
        --output tsv)

    export TASKHUB="my-taskhub"
    ```

1.  Run the application
    ```bash
    cd samples/durable-task-sdks/dotnet/ScheduleWebApp
    dotnet run
    ```
    The application will start and listen on `http://localhost:5000` by default.

## Identity-based authentication

Learn how to set up [identity-based authentication](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity?tabs=df&pivots=az-cli) when you deploy the app Azure.  

## API Endpoints

The sample provides a list of schedule management endpoints that can be found at [`samples/durable-task-sdks/dotnet/ScheduleWebApp/ScheduleWebApp.http`](ScheduleWebApp.http).
The file contains sample REST requests for testing the schedule management endpoints. It includes examples for creating, retrieving, listing, updating, pausing, resuming, and deleting schedules. These examples are designed for use with REST client tools (e.g., the VS Code REST Client extension).

## Schedule Creation Options
A schedule defines how and when an orchestration is triggered on a recurring basis. In this sample, the schedule is used to automatically invoke the CacheClearingOrchestrator at regular intervals. The ScheduleCreationOptions object includes:
- id: A unique identifier for the schedule.
- orchestrationName: The name of the orchestrator that the schedule will trigger.
- interval: The recurring frequency (in TimeSpan format, e.g., hh:mm:ss) for the orchestrator invocation.
- orchestrationInput: Optional data passed to the orchestrator when it runs.
- startAt/endAt: Optional properties to set a specific start or end time for the recurrence.
- startImmediatelyIfLate: When set to true, ensures that if the schedule has missed its trigger time, it will run as soon as possible.
  
## Monitoring

You can monitor your scheduled tasks using the Durable Task Scheduler dashboard. Get the dashboard URL using:

```bash
az durabletask taskhub show \
    --resource-group my-resource-group \
    --scheduler-name my-scheduler \
    --name "schedule-webapp" \
    --query "properties.dashboardUrl" \
    --output tsv
```

## Configuration

The application uses the following configuration:

- `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` - Connection string for durable task scheduler
- `ASPNETCORE_ENVIRONMENT` - Application environment (Development/Production)

These can be configured through:
- Environment variables
- User secrets (during development)
- appsettings.json

## Additional Resources

- [Durable Task SDK Documentation](https://github.com/microsoft/durabletask-dotnet)
- [Durable task scheduler Documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
