# Orchestration Versioning with the Durable Task SDK for .NET

In addition to [Durable Functions](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview), the [Durable Task SDK for .NET](https://github.com/microsoft/durabletask-dotnet) can also use the durable task scheduler for managing orchestration state.

This directory includes a sample .NET console app that demonstrates how to utilize Orchestration Versioning with the Durable Task SDK for .NET.

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
- [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed


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
    docker run -it -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
    ```
    Wait a few seconds for the container to be ready.

1. Set the scheduler connection string:
    ```bash
    export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
    ```

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

    Note that it may take a minute for the role assignment to take effect.

1. Generate a connection string for the scheduler and task hub resources and save it to the `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` environment variable:

    ```bash
    endpoint=$(az durabletask scheduler show \
        --resource-group my-resource-group \
        --name my-scheduler \
        --query "properties.endpoint" \
        --output tsv)
    taskhub="my-taskhub"
    export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=$endpoint;TaskHub=$taskhub;Authentication=DefaultAzure"
    ```

    The `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` environment variable is used by the sample app to connect to the durable task scheduler resources. The type of credential to use is specified by the `Authentication` segment. Supported values include `DefaultAzure`, `ManagedIdentity`, `WorkloadIdentity`, `Environment`, `AzureCLI`, and `AzurePowerShell`.

## Orchestration Versioning

A key issue in orchestration based workflows is handling different versions of orchestrations. Without any functionality built around versioning, the issue occurs when the number or order of tasks is changed in an orchestration. During the replay of events (for any inprogress orchestrations), the history will not match and a `NonDeterministicError` will be returned as the orchestration fails. The .NET portable SDK for Durable Task now offers several options for how to handle versioning in orchestrations.

### Client/Context Based Conditional Versioning

In order for an orchestration to have a version, it must first be set in the client. For the .NET SDK, this is done through the standard host builder extensions, as seeen below:

```csharp
builder.Services.AddDurableTaskClient(builder =>
{
    builder.UseDurableTaskScheduler(connectionString);
    builder.UseDefaultVersion("1.0.0");
});
```

Once that is added, any orchestration started by this host will use the version `1.0.0`. The version itself is a simple string and accepts any value. However, the SDK will try and convert it to .NET's `System.Version`. If it can be converted, that library is used for comparison, if not, a simple string comparison is used.

By supplying the version in the client, it also becomes available in the `TaskOrchestrationContext`. This means the version can be used in conditional statements. So, as long as newer versions of an orchestration have the appropriate version gating, both the old and the new version so of the orchestration can run together on the same host. An example of how the version can be used can be seen below:


```csharp
[DurableTask]
class HelloCities : TaskOrchestrator<string, List<string>>
{
    private readonly string[] Cities = ["Seattle", "Amsterdam", "Hyderabad", "Kuala Lumpur", "Shanghai", "Tokyo"];

    public override async Task<List<string>> RunAsync(TaskOrchestrationContext context, string input)
    {
        List<string> results = [];
        foreach (var city in Cities)
        {
            results.Add(await context.CallSayHelloAsync($"{city} v{context.Version}"));
            if (context.CompareVersionTo("2.0.0") >= 0)
            {
                results.Add(await context.CallSayGoodbyeAsync($"{city} v{context.Version}"));
            }
        }

        Console.WriteLine("HelloCities orchestration completed.");
        return results;
    }
}
```

In this example, we've added a `SayGoodbye` activity to the `HelloCities` orchestration. This is only called if the orchestration is at least version `2.0.0`. With the simple conditional statement, any orchestration with a version less than `2.0.0` will continue to function and any new orchestration will have the new activity in it.

### Worker Based Versioning

Another strategy that can be used for handling versions is setting up worker versioning. Orchestrations will still need a client version in order to have the version set, but this method allows the user to avoid conditionals in their orchestrations. Worker versioning allows the worker itself to choose how to act on different version so of orchestrations before those orchestrations start executing. Worker versioning has several fields to set, which are detailed below:

1. The version of the worker itself
2. The strategy that the worker will use to match against the orchestration's version
3. The strategy that the worker should take if the version does not meet the matching strategy

The different match strategies are as follows:

| Name           | Description                                                                              |
|----------------|------------------------------------------------------------------------------------------|
| None           | The version is not considered when work is being processed                               |
| Strict         | The version in the orchestration and the worker must match exactly                       |
| CurrentOrOlder | The version in the orchestration must be equal to or less than the version in the worker |

The different failure strategies are as follows:

| Name   | Description                                                                                               |
|--------|-----------------------------------------------------------------------------------------------------------|
| Reject | The orchestration will be rejected by the worker but remain in the work queue to be attempted again later |
| Fail   | The orchestration will be failed and removed from the work queue                                          |

Similar to the client versioning, these are all set via the standard host builder pattern:

```csharp
builder.Services.AddDurableTaskWorker(builder =>
{
    builder.AddTasks(r => r.AddAllGeneratedTasks());
    builder.UseDurableTaskScheduler(connectionString);
    builder.UseVersioning(new DurableTaskWorkerOptions.VersioningOptions
    {
        Version = "1.0.0",
        MatchStrategy = DurableTaskWorkerOptions.VersionMatchStrategy.Strict,
        FailureStrategy = DurableTaskWorkerOptions.VersionFailureStrategy.Reject,
    });
});
```

## Running the sample

In the same terminal window as above, use the following steps to run the sample on your local machine.

1. Clone this repository.

1. Open a terminal window and navigate to the `samples/durable-task-sdks/dotnet/OrchestrationVersioning` directory.

1. Run the following command to build and run the sample:

    ```bash
    dotnet run
    ```

You should see output similar to the following:

```plaintext
2025-01-14T22:31:10.926Z info: Microsoft.DurableTask[1] Durable Task gRPC worker starting.
2025-01-14T22:31:11.041Z info: Microsoft.Hosting.Lifetime[14] Now listening on: http://localhost:5008
2025-01-14T22:31:11.042Z info: Microsoft.Hosting.Lifetime[0] Application started. Press Ctrl+C to shut down.
2025-01-14T22:31:11.043Z info: Microsoft.Hosting.Lifetime[0] Hosting environment: Development
2025-01-14T22:31:11.043Z info: Microsoft.Hosting.Lifetime[0] Content root path: /home/cgillum/code/github.com/Azure/Azure-Functions-Durable-Task-Scheduler-Private-Preview/samples/durable-task-sdks/dotnet/OrchestrationVersioning
2025-01-14T22:31:14.885Z info: Microsoft.DurableTask[4] Sidecar work-item streaming connection established.
```

Now, the ASP.NET Web API is running locally on your machine, and any output from the app will be displayed in the terminal window.

To run orchestrations, you can use a tool like [Postman](https://www.postman.com/) or [curl](https://curl.se/) in another terminal window to send a POST request to the `/scenarios/hellocities?count=N` endpoint, where `N` is the number of orchestrations to start.

```bash
curl -X POST "http://localhost:5008/scenarios/hellocities?count=10"
```

You should then see output in the ASP.NET Web App terminal window showing the logs associated with the orchestrations that were started.

## View orchestrations in the dashboard

You can view the orchestrations in the Durable Task Scheduler dashboard by navigating to the scheduler-specific dashboard URL in your browser.

Use the following PowerShell command from a new terminal window to get the dashboard URL:

**PowerShell**:

```powershell
$dashboardUrl = az durabletask taskhub show `
    --resource-group "my-resource-group" `
    --scheduler-name "my-scheduler" `
    --name "my-taskhub" `
    --query "properties.dashboardUrl" `
    --output tsv
$dashboardUrl
```

**Bash**:

```bash
dashboardUrl=$(az durabletask taskhub show \
    --resource-group "my-resource-group" \
    --scheduler-name "my-scheduler" \
    --name "my-taskhub" \
    --query "properties.dashboardUrl" \
    --output tsv)
echo $dashboardUrl
```

The URL should look something like the following:

```plaintext
https://dashboard.durabletask.io/subscriptions/{subscriptionID}/schedulers/my-scheduler/taskhubs/my-taskhub?endpoint=https%3a%2f%2fmy-scheduler-gvdmebc6dmdj.northcentralus.durabletask.io
```

Once logged in, you should see the orchestrations that were created by the sample app. Below is an example of what the dashboard might look like (note that some of the details will be different than the screenshot):

![Durable Task Scheduler dashboard](../../../../media/images/durable-task-sdks/portable-sample-dashboard.png)

## Optional: Deploy to Azure Container Apps

1. Create an container app following the instructions in the [Azure Container App documentation](https://learn.microsoft.com/azure/container-apps/get-started?tabs=bash).
1. During step 1, specify the deployed container app code folder at samples\durable-task-sdks\dotnet\OrchestrationVersioning
1. Follow the instructions to create a user managed identity and assign the `Durable Task Data Contributor` role then attach it to the container app you created in step 1 at [Azure-Functions-Durable-Task-Scheduler-Private-Preview](..\..\..\..\docs\configure-existing-app.md#run-the-app-on-azure-net). Please skip section "Add required environment variables to app" since these environment variables are not required for deploying to container app.
1. Call the container app endpoint at `http://sampleapi-<your-container-app-name>.azurecontainerapps.io/scenarios/hellocities?count=10`, Sample curl command:

    ```bash
    curl -X POST "https://sampleapi-<your-container-app-name>.azurecontainerapps.io/scenarios/hellocities?count=10"
    ```

1. You should see the orchestration created in the Durable Task Scheduler dashboard.
