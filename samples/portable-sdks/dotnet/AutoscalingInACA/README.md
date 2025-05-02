# Autoscaling in Azure Container Apps Pattern

## Description of the Sample

This sample demonstrates how to implement autoscaling with the Azure Durable Task Scheduler using the .NET SDK in Azure Container Apps. The pattern showcases an orchestration workflow that can benefit from dynamically scaling worker instances based on load.

In this sample:
1. The orchestrator calls the `SayHelloActivity` which greets the user with their name
2. The result is passed to the `ProcessGreetingActivity` which adds to the greeting
3. The result is then passed to the `FinalizeResponseActivity` which completes the greeting
4. The final greeting message is returned to the client

This pattern is useful for:
- Creating scalable workflows that can handle varying loads
- Efficiently utilizing resources by scaling container instances up or down
- Building resilient systems that can respond to increased demand
- Implementing cost-effective processing for variable workloads

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for building the image)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

### Deploying with Azure Developer CLI (AZD)

This sample includes an `azure.yaml` configuration file that allows you to deploy the entire solution to Azure using Azure Developer CLI (AZD). This deployment includes a custom KEDA scaler for your Container Apps that automatically scales based on the Durable Task Scheduler's workload.

#### Prerequisites for AZD Deployment

1. Install [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
2. Authenticate with Azure:
   ```bash
   azd auth login
   ```

#### Deployment Steps

1. Navigate to the AutoscalingInACA sample directory:
   ```bash
   cd /path/to/Durable-Task-Scheduler/samples/portable-sdks/dotnet/AutoscalingInACA
   ```

2. Initialize the AZD environment with a unique name:
   ```bash
   azd init --template .
   ```

3. Provision resources and deploy the application:
   ```bash
   azd up
   ```
   This command will:
   - Provision Azure resources (including Azure Container Apps and Durable Task Scheduler)
   - Build and deploy both the Client and Worker components
   - Set up the custom KEDA scaler for the Worker container app
   - Configure the necessary connections between components

4. Monitor your orchestrations using the Azure Portal by navigating to your Durable Task Scheduler resource dashboard, or view the container app logs using the Azure Portal:

   - Navigate to the Azure Portal and find your resource group
   - Go to each Container App (Worker and Client) that was deployed
   - In the left sidebar, select "Log stream" under the "Monitoring" section
   - Observe the logs in real-time to confirm that:
     - Client app is submitting orchestration requests
     - Worker app is processing the orchestrations through each activity
     - You should see logs showing the completion of `SayHelloActivity`, `ProcessGreetingActivity`, and `FinalizeResponseActivity`
   - This confirms that your orchestrations are being processed correctly

## Understanding the Custom Scaler

This sample implements autoscaling using KEDA (Kubernetes Event-Driven Autoscaling) with a custom scaler for Azure Container Apps. The scaler monitors the Durable Task Scheduler workload and automatically adjusts the number of worker instances based on the current orchestration load.

### How the Custom Scaler Works

The custom scaler:
- Monitors the number of pending orchestrations in the task hub
- Scales the number of worker replicas up when there is increased workload
- Scales back down when the load decreases
- Provides efficient resource utilization by matching capacity to demand

### Confirming the Scaler is Working

To verify that the autoscaling is functioning correctly:

1. Navigate to the Azure Portal and find your resource group
2. Go to the Worker Container App that was deployed
3. Select the "Revision management" section in the sidebar
4. Observe the "Replica count" as it changes based on load
5. You can also check the "Scale" tab to see the KEDA scaler configuration

To test the autoscaling:
1. Run the client with a large number of orchestration requests
   ```bash
   # You can modify the Client/Program.cs to schedule more orchestrations
   # Then run from your AZD environment:
   azd deploy --service client
   ```
2. Monitor the replica count in the Azure Portal
3. You should see the number of replicas increase as the workload grows
4. Once the orchestrations complete, the replicas should scale back down after a cooldown period

You can also view the Application Insights logs to see the scaling events and performance metrics for your Container Apps.
