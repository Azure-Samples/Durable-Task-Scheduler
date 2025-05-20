# Funds Transfer with the Durable Task SDK for .NET

The [Durable Task SDK for .NET](https://github.com/microsoft/durabletask-dotnet) supports [Durable Entities](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities) when used with the Durable Task Scheduler. This sample shows how to use Durable Entities when targeting the Durable Task Scheduler service, with no Azure Functions dependency.

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
- [Docker](https://www.docker.com/get-started)

## Running the sample with the Durable Task Scheduler Emulator

The sample app can be run locally using the Durable Task Scheduler Emulator. The emulator is a containerized version of the Durable Task Scheduler service that persists state in memory. It is useful for development and testing purposes.

From a terminal window as above, use the following steps to run the sample on your local machine.

1. Clone this repository.

1. Navigate to the `samples/durable-task-sdks/dotnet/EntitiesSample` directory.

1. Start the Durable Task Scheduler Emulator.

    ```bash
    docker pull mcr.microsoft.com/dts/dts-emulator:latest
    ```

    ```bash
    docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest    
    ```

1. Set the `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` environment variable:

    ```bash
    export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
    ```

1. Run the following command to build and run the sample:

    ```bash
    dotnet run
    ```

    You should see output similar to the following:

    ```plaintext
    2025-02-25T04:11:48.609Z info: Microsoft.DurableTask[1] Durable Task gRPC worker starting and connecting to localhost:8080.
    2025-02-25T04:11:48.682Z info: Microsoft.Hosting.Lifetime[14] Now listening on: http://localhost:5203
    2025-02-25T04:11:48.683Z info: Microsoft.Hosting.Lifetime[0] Application started. Press Ctrl+C to shut down.
    2025-02-25T04:11:48.684Z info: Microsoft.Hosting.Lifetime[0] Hosting environment: Development
    2025-02-25T04:11:48.684Z info: Microsoft.Hosting.Lifetime[0] Content root path: C:\GitHub\Azure-Functions-Durable-Task-Scheduler-Private-Preview\samples\durable-task-sdks\dotnet\EntitiesSample
    2025-02-25T04:11:48.747Z info: Microsoft.DurableTask[4] Sidecar work-item streaming connection established.
    ```

    Now, the ASP.NET Web API is running locally on your machine, connected to the local emulator. Any output from the app will be displayed in the terminal window.

1. Initialize an account with ID `123` with a balance of `100`:

    ```bash
    curl -X POST "http://localhost:5203/accounts/123/deposit" -d '{"amount": 100}' -H "Content-Type: application/json" -i
    ```

    The implementation of this API signals an account entity with ID `123` to deposit the funds. An HTTP 202 Accepted response should be returned.

1. Initialize a second account with ID `456` with a balance of `100`:

    ```bash
    curl -X POST "http://localhost:5203/accounts/456/deposit" -d '{"amount": 100}' -H "Content-Type: application/json" -i
    ```

    The implementation of this API signals an account entity with ID `456` to deposit the funds. An HTTP 202 Accepted response should be returned.

1. Transfer $50 from account `123` to account `456`:

    ```bash
    curl -X POST "http://localhost:5203/accounts/transfers" -d '{"sourceId":"123","destinationId":"456","amount": 50}' -H "Content-Type: application/json"
    ```

    The implementation of this API starts an orchestration that transfers funds from account `123` to account `456`. An HTTP 202 Accepted response should be returned with a JSON payload that looks like the following:

    ```json
    {
        "transactionId": "4c51db815309413b92517cc7cbfb81e6"
    }
    ```

    The actual `transactionId` will be different because it is randomly generated. This transaction ID corresponds to the orchestration instance ID. Make a note of this value for the next step.

1. After a few seconds, check the status of the transfer. Replace the `transactionId` with the one returned in the previous step:

    ```bash
    curl -X GET "http://localhost:5203/accounts/transfers/{transactionId}"
    ```

    The implementation of this API queries the status of the orchestration with the given ID. An HTTP 200 OK response should be returned with a JSON payload that looks like the following:

    ```json
    {
        "transactionId": "4c51db815309413b92517cc7cbfb81e6",
        "initiatedAt": "2025-02-25T05:01:53",
        "status": "Completed",
        "transferResult": "Transferred"
    }
    ```

1. Confirm the balance of account `123`:

    ```bash
    curl -X GET "http://localhost:5203/accounts/123/balance"
    ```

    The implementation of this API queries the state of the account entity with ID `123`. An HTTP 200 OK response should be returned with a JSON payload that looks like the following:

    ```json
    {
        "accountId": "123",
        "balance": 50
    }
    ```

1. Confirm the balance of account `456`:

    ```bash
    curl -X GET "http://localhost:5203/accounts/456/balance"
    ```

    The implementation of this API queries the state of the account entity with ID `456`. An HTTP 200 OK response should be returned with a JSON payload that looks like the following:

    ```json
    {
        "accountId": "456",
        "balance": 150
    }
    ```

## View orchestrations and entities in the dashboard

You can view the sample's orchestrations and entities in the Durable Task Scheduler emulator's dashboard by navigating to `http://localhost:8082` in your browser and selecting the `default` task hub.
