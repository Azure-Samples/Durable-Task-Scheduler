# Durable Functions SAGA Pattern with Durable Task Scheduler

## What is the SAGA Pattern?

The SAGA pattern is a design pattern for managing long-running business transactions where traditional ACID transactions across multiple operations are impractical. The SAGA pattern offers an elegant approach by:

1. Breaking down a complex transaction into a sequence of smaller, local transactions
2. Providing compensating transactions to undo changes if any step fails
3. Ensuring eventual consistency across the system

### Why SAGA is Important

In complex transaction processing systems:
- Long-running operations need to be broken into manageable steps
- Multiple resources might need to be updated atomically
- Failures must be handled gracefully to maintain data integrity

The SAGA pattern addresses these challenges by providing a framework for coordinating a sequence of operations while maintaining data consistency through compensation mechanisms.

## Implementing SAGA with Durable Functions and Durable Task Scheduler

This sample demonstrates implementing the SAGA pattern using:
- **Azure Durable Functions**: A serverless orchestration framework that handles state management
- **Durable Task Scheduler**: A backend service that manages the execution of durable tasks

## Order Processing Workflow

This sample implements an order processing workflow that demonstrates the SAGA pattern through these steps:

1. Send order notification
2. Reserve inventory
3. Request approval
4. Process payment
5. Update inventory
6. Process delivery

If any step fails, appropriate compensation actions are automatically executed:
- Payment failure → Release inventory reservation
- Inventory update failure → Refund payment + Release inventory
- Delivery failure → Refund payment + Restore inventory

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                              Azure Function App                                  │
│                                                                                  │
│  ┌────────────────┐     ┌────────────────────────────────────────────┐          │
│  │                │     │                                            │          │
│  │  HTTP Triggers │     │       OrderProcessingOrchestrator          │          │
│  │  ┌──────────┐  │     │                                            │          │
│  │  │ Start    │  │     │  ┌─────────┐  ┌─────────┐  ┌─────────┐     │          │
│  │  │ Order    ├──┼─────┼─►│ Notify  ├─►│ Reserve ├─►│ Approval├─┐   │          │
│  │  └──────────┘  │     │  └─────────┘  └─────────┘  └─────────┘ │   │          │
│  │  ┌──────────┐  │     │                                       │   │          │
│  │  │ Check    │  │     │  ┌─────────┐  ┌─────────┐  ┌─────────┐ │   │          │
│  │  │ Status   │  │     │  │ Process │◄─┤ Update  │◄─┤ Delivery│◄┘   │          │
│  │  └──────────┘  │     │  │ Payment │  │ Inventory│  │         │     │          │
│  │  ┌──────────┐  │     │  └────┬────┘  └────┬────┘  └────┬────┘     │          │
│  │  │Terminate │  │     │       │            │            │          │          │
│  │  │Order     │  │     │  ┌────▼────┐  ┌────▼────┐  ┌────▼────┐     │          │
│  │  └──────────┘  │     │  │ Refund  │  │ Restore │  │ Cancel  │     │          │
│  │                │     │  │ Payment │  │ Inventory│  │ Delivery│     │          │
│  └────────────────┘     │  └─────────┘  └─────────┘  └─────────┘     │          │
│                         │           Compensation Actions             │          │
│                         └────────────────────────────────────────────┘          │
│                                         │                                       │
└─────────────────────────────────────────┼───────────────────────────────────────┘
                                          │
                                          ▼
                 ┌─────────────────────────────────────────────┐
                 │         Durable Task Scheduler              │
                 │                                             │
                 │  ┌──────────────┐     ┌──────────────────┐  │
                 │  │Task Execution│     │ Orchestration    │  │
                 │  │  Engine      │     │ State Management │  │
                 │  └──────────────┘     └──────────────────┘  │
                 │           │                   │             │
                 └───────────┼───────────────────┼─────────────┘
                             │                   │
                             ▼                   ▼
                  ┌──────────────────────────────────────┐
                  │          Storage Provider            │
                  │   (Azure Storage or Local Emulator)  │
                  └──────────────────────────────────────┘
```

## Components

- **HTTP Triggers**: REST endpoints to start and manage workflow instances
- **Orchestrator**: Coordinates the overall workflow and handles failures
- **Activity Functions**: Individual stateless steps in the workflow
- **Durable Task Scheduler**: Backend service managing orchestration state

## Project Structure

```
DurableFunctionsSaga/
├── Activities/                 # Individual activity functions
│   ├── ApprovalActivities.cs   # Order approval
│   ├── DeliveryActivities.cs   # Delivery processing (with demo failures)
│   ├── InventoryActivities.cs  # Inventory operations and compensation
│   ├── NotificationActivities.cs # Notifications
│   └── PaymentActivities.cs    # Payment processing and refunds
├── Functions/
│   └── HttpTriggers.cs         # REST endpoints for the SAGA
├── Models/                     # Data models
│   ├── Order.cs, Payment.cs, Inventory.cs, etc.
├── Orchestrators/
│   └── OrderProcessingOrchestrator.cs  # Main SAGA implementation
├── Program.cs                  # Application startup and DI configuration
├── host.json                   # Function app and Durable Task config
└── local.settings.json         # Local settings including connection strings
```

Each component plays a specific role:

1. **Orchestrator** - Defines the workflow sequence and compensation logic
2. **Activity Functions** - Perform individual steps and compensating actions
3. **HTTP Triggers** - Provide the external interface to start and monitor workflows
4. **Models** - Define the data structures passed between activities

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Durable Task Scheduler Emulator](https://github.com/microsoft/durabletask-dotnet) (for local development)
- [Azurite](https://azure.microsoft.com/features/storage-explorer/) (For functions runtime)

## Running with Durable Task Scheduler

The sample is configured to use the Durable Task Scheduler as the backend for Durable Functions:

### Configuring Durable Task Scheduler
There are two ways to run this sample locally:

#### Using the Emulator (Recommended)
The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. **Pull the Docker Image for the Emulator**:
   ```bash
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. **Run the Emulator**:
   ```bash
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```
   
   Wait a few seconds for the container to be ready.

   Note: The example code automatically uses the default emulator settings (endpoint: http://localhost:8080, taskhub: default). You don't need to set any environment variables.

3. **Update Configuration** (if needed):
   Verify that `local.settings.json` includes the Durable Task Scheduler connection:
   ```json
   {
     "IsEncrypted": false,
     "Values": {
       "AzureWebJobsStorage": "UseDevelopmentStorage=true",
       "FUNCTIONS_WORKER_RUNTIME": "dotnet-isolated",
       "DurableTaskSchedulerConnection": "Endpoint=http://localhost:8080;Authentication=None"
     },
     "Host": {
       "LocalHttpPort": 7071
     }
   }
   ```

   And check that `host.json` is configured to use the Durable Task Scheduler:
   ```json
   {
     "version": "2.0",
     "extensions": {
       "durableTask": {
         "hubName": "default",
         "storageProvider": {
           "type": "AzureManaged",
           "connectionStringName": "DurableTaskSchedulerConnection"
         }
       }
     }
   }
   ```

4. **Build and Run the Project**:
   ```bash
   dotnet build
   func start
   ```

5. **Test the Order Processing Workflow**:

   You can use the included `test.http` file if you have the REST Client extension in VS Code, or use curl commands:
   
   ```bash
   # Start a new order processing saga
   curl -X POST http://localhost:7071/api/orders \
     -H "Content-Type: application/json" \
     -d '{"customerId": "customer123", "productId": "product456", "quantity": 5, "amount": 100.00}'
   
   # Check the status (replace {instanceId} with the ID from the response)
   curl http://localhost:7071/api/orders/{instanceId}
   
   # Terminate an order (if needed)
   curl -X POST http://localhost:7071/api/orders/{instanceId}/terminate
   ```
   
   The Durable Task Scheduler dashboard will be available at http://localhost:8082 where you can monitor all orchestration instances.

6. **Monitor Orchestrations in the Dashboard**:
   Open http://localhost:8082 in your browser to see the Durable Task Scheduler dashboard.
   Here you can monitor orchestrations, activities, and review execution history.

7. **Observe the Compensation Mechanism**:
   The sample is configured to randomly fail at the delivery step (50% chance).
   When a failure occurs, watch the logs and dashboard to see compensation actions being executed.



## Configuration

The project uses AzureStorage as the Durable Functions provider:

```json
// local.settings.json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "dotnet-isolated"
  }
}
```

For production, set your Azure Storage connection string:
```bash
export AzureWebJobsStorage="DefaultEndpointsProtocol=https;AccountName=<account>;AccountKey=<key>;EndpointSuffix=core.windows.net"
```

## API Endpoints

- **POST /api/orders**: Start a new order processing workflow
  ```json
  // Request body
  {
    "customerId": "customer123",
    "productId": "product456",
    "quantity": 5,
    "amount": 100.00
  }

  // Response
  {
    "id": "abc123def456",                               // Orchestration instance ID
    "orderId": "58678b9e-3225-46ad-b9e6-101c52216216", // Generated order ID
    "statusQueryGetUri": "http://localhost:7071/api/orders/abc123def456",
    "terminatePostUri": "http://localhost:7071/api/orders/abc123def456/terminate"
  }
  ```

- **GET /api/orders/{instanceId}**: Check order status
  ```json
  // Response
  {
    "id": "abc123def456",
    "runtimeStatus": "Completed", // Or "Running", "Failed", etc.
    "input": { /* Order details */ },
    "output": { /* Result (if completed) */ },
    "createdTime": "2025-06-04T20:24:04.710714+00:00",
    "lastUpdatedTime": "2025-06-04T20:24:05.941042+00:00"
  }
  ```

- **POST /api/orders/{instanceId}/terminate**: Terminate a running workflow

## Key Features

- **Intentional Failures**: The delivery step fails randomly (50% chance) to demonstrate compensation
- **Compensation Logic**: When failures occur, previous steps are automatically reversed
- **Stateful Workflow**: State is maintained throughout the orchestration

## Code Example

```csharp
// From DeliveryActivities.cs - Demonstrates intentional failure
if (DateTime.UtcNow.Second % 2 == 0) // Fail 50% of the time
{
    _logger.LogError("Delivery service is currently unavailable");
    throw new Exception("Delivery service is currently unavailable");
}

// From OrderProcessingOrchestrator.cs - Compensation actions
catch (Exception ex) {
    _logger.LogError(ex, "Delivery failed. Initiating compensation.");
    await context.CallActivityAsync("RefundPaymentActivity", payment);
    await context.CallActivityAsync("RestoreInventoryActivity", inventory);
    order.Status = "Failed - Delivery Error";
}
```

## Resources

- [Durable Functions Documentation](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
- [SAGA Pattern](https://docs.microsoft.com/azure/architecture/reference-architectures/saga/saga)
