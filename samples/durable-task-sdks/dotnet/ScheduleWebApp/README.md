# Schedule Orchestrations with Durable Task SDK for .NET

This sample demonstrates a web application that leverages the durable task scheduler to manage recurring background tasks. Users can create, update, pause, resume, and delete schedules through REST endpoints provided by the ScheduleController. A dedicated orchestrator (CacheClearingOrchestrator) is used to perform periodic operations, such as clearing caches, by automatically triggering the scheduled tasks at configured intervals.

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
- Azure subscription 

## Setup

1. Install the durable task scheduler CLI extension:
   ```bash
   az upgrade
   az extension add --name durabletask --allow-preview true
   ```

2. Create and configure Azure resources:
   ```bash
   # Create resource group
   az group create --name my-resource-group --location northcentralus

   # Create scheduler
   az durabletask scheduler create \
       --resource-group my-resource-group \
       --name my-scheduler \
       --ip-allowlist '["0.0.0.0/0"]' \
       --sku-name "Dedicated" \
       --sku-capacity 1

   # Create task hub
   az durabletask taskhub create \
       --resource-group my-resource-group \
       --scheduler-name my-scheduler \
       --name "schedule-webapp"
   ```

3. Grant permissions to your account:
   ```bash
   subscriptionId=$(az account show --query "id" -o tsv)
   loggedInUser=$(az account show --query "user.name" -o tsv)

   az role assignment create \
       --assignee $loggedInUser \
       --role "Durable Task Data Contributor" \
       --scope "/subscriptions/$subscriptionId/resourceGroups/my-resource-group/providers/Microsoft.DurableTask/schedulers/my-scheduler/taskHubs/schedule-webapp"
   ```

4. Configure the connection string:

   **PowerShell**:
   ```powershell
   $endpoint = az durabletask scheduler show `
       --resource-group my-resource-group `
       --name my-scheduler `
       --query "properties.endpoint" `
       --output tsv
   $taskhub = "schedule-webapp"
   $env:DURABLE_TASK_SCHEDULER_CONNECTION_STRING = "Endpoint=$endpoint;TaskHub=$taskhub;Authentication=DefaultAzure"
   ```

   **Bash**:
   ```bash
   endpoint=$(az durabletask scheduler show \
       --resource-group my-resource-group \
       --name my-scheduler \
       --query "properties.endpoint" \
       --output tsv)
   taskhub="schedule-webapp"
   export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=$endpoint;TaskHub=$taskhub;Authentication=DefaultAzure"
   ```

## Running the Application

1. Navigate to the application directory:
   ```bash
   cd samples/durable-task-sdks/dotnet/ScheduleWebApp
   ```

2. Run the application:
   ```bash
   dotnet run
   ```

The application will start and listen on `http://localhost:5000` by default.

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
