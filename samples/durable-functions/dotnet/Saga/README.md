# Order Processing with Saga Pattern in .NET 8 Durable Functions

This project demonstrates an implementation of the Saga pattern using .NET 8 and Azure Durable Functions. The Saga pattern is a failure management pattern that helps maintain data consistency across distributed business processes by defining compensating actions for each step in the workflow.

## Overview

The sample implements an order processing workflow with multiple steps that must either all complete successfully or be rolled back entirely. This ensures data consistency across systems even when failures occur.

## Order Processing Flow

Our order processing saga handles a complete e-commerce order scenario:

1. **Send notification** - Inform the customer that order processing is starting
2. **Reserve inventory** - Mark inventory as reserved for this order
3. **Request approval** - Get order approval (may be manual or automated)
4. **Process payment** - Handle payment processing
5. **Update inventory** - Convert reserved inventory to confirmed
6. **Process delivery** - Schedule delivery of the order

If any step fails, the system automatically executes compensation actions in reverse order to maintain consistency.

## The Saga Pattern Implementation

The key to our implementation is the `Compensations` class, which:

```csharp
public class Compensations
{
    private readonly List<Func<Task>> _compensations = new();
    
    // Register a compensation action to be executed if the workflow fails
    public void AddCompensation<T>(string activityName, T input) { ... }
    
    // Execute all registered compensation actions in reverse order
    public async Task CompensateAsync(bool inParallel = false) { ... }
}
```

### Core Principles

1. **Register Compensation First**: Always register a compensation action before performing the actual operation
2. **LIFO Execution Order**: Compensations are executed in reverse order of registration
3. **Error Handling**: Failures during compensation don't stop other compensations
4. **Optional Parallel Execution**: Compensations can run in parallel if desired

### Compensation Actions

For each business operation, we define a corresponding compensation:

| Operation | Compensation |
|-----------|-------------|
| Reserve Inventory | Release Inventory |
| Process Payment | Refund Payment |
| Update Inventory | Restore Inventory |

## Benefits of This Approach

1. **Data Consistency**: Ensures all systems remain consistent even during failures
2. **Clear Separation of Concerns**: Main workflow logic is separate from compensation logic  
3. **Reduced Complexity**: Eliminates deeply nested try/catch blocks
4. **Automatic Management**: System tracks and executes compensations
5. **Durability**: State is persisted automatically by Durable Functions

## Running the Sample Locally

### Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
2. [Azure Functions Core Tools](https://docs.microsoft.com/azure/azure-functions/functions-run-local) (v4.x)
3. [Docker](https://www.docker.com/products/docker-desktop/) for running the DTS emulator

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

### Setup and Run

1. **Clone the repository**:
   ```
   git clone <repository-url>
   cd durable-functions/dotnet/Saga
   ```

2. **Start the function app**:
   ```
   func start
   ```

4. **Test the workflow**:
   - Use the provided `test.http` file if you have the REST Client extension in VS Code
   - Or use curl/Postman to make the following requests:

   ```
   # Start a new order processing workflow
   POST http://localhost:7071/api/orders
   Content-Type: application/json

   {
     "customerId": "customer123",
     "productId": "product456",
     "quantity": 5,
     "amount": 100.00
   }

   # The response will include an instanceId - copy it for the next request

   # Check the status
   GET http://localhost:7071/api/orders/{instanceId}
   ```

5. **Monitor the logs**:
   - Watch the console output to see the workflow execution
   - In approximately 50% of runs, the delivery step will fail (by design)
   - The approval step also fails 25% of the time (by design)
   - When it fails, you'll see the compensation actions run in reverse order

## Project Structure

- **Saga/Compensations.cs** - Core compensation management logic
- **Models/** - Data models for order, inventory, payment, etc.
- **Activities/** - Implementation of business operations and compensations
- **Orchestrators/OrderProcessingOrchestrator.cs** - The main workflow orchestrator
- **Functions/HttpTriggers.cs** - API endpoints to interact with the workflow

## API Reference

- **POST /api/orders** - Start a new order processing workflow
  - Body: Order details (customerId, productId, quantity, amount)
  - Returns: Instance ID and status URL

- **GET /api/orders/{instanceId}** - Get status of an order processing workflow
  - Returns: Current state, status, and output

- **POST /api/orders/{instanceId}/terminate** - Cancel a running workflow
  - Returns: Confirmation of termination
