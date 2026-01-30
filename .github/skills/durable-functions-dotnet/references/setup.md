# Durable Functions Setup Reference

Complete setup and deployment guidance for Azure Durable Functions with Durable Task Scheduler.

## Local Development

### Prerequisites

```bash
# Install Azure Functions Core Tools
brew tap azure/functions
brew install azure-functions-core-tools@4

# Install .NET SDK
brew install dotnet

# Install Azure CLI (optional, for Azure deployment)
brew install azure-cli

# Install Azurite (Azure Storage emulator)
npm install -g azurite
```

### Start Local Emulator

```bash
# Terminal 1: Start Azurite (required for Azure Functions)
azurite start

# Terminal 2: Start Durable Task Scheduler emulator
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d -p 8080:8080 -p 8082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest

# Dashboard available at http://localhost:8082
```

### Docker Compose (All-in-One)

```yaml
# docker-compose.yml
version: '3.8'

services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"  # Blob
      - "10001:10001"  # Queue
      - "10002:10002"  # Table
    volumes:
      - azurite-data:/data
    command: azurite --blobHost 0.0.0.0 --queueHost 0.0.0.0 --tableHost 0.0.0.0

  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"    # gRPC/HTTP endpoint
      - "8082:8082"    # Dashboard
    environment:
      - DTS_EMULATOR_LOG_LEVEL=Information
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  azurite-data:
```

```bash
docker-compose up -d
```

## Project Setup

### Create New Project

```bash
# Create Functions project
func init MyDurableFunctions --worker-runtime dotnet-isolated --target-framework net8.0

cd MyDurableFunctions

# Add required packages
dotnet add package Microsoft.Azure.Functions.Worker.Extensions.DurableTask
dotnet add package Azure.Identity

# Optional: class-based orchestrations
dotnet add package Microsoft.DurableTask.Generators
```

### Complete .csproj Template

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <AzureFunctionsVersion>v4</AzureFunctionsVersion>
    <OutputType>Exe</OutputType>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <RootNamespace>MyDurableFunctions</RootNamespace>
  </PropertyGroup>
  
  <ItemGroup>
    <PackageReference Include="Microsoft.Azure.Functions.Worker" Version="2.*" />
    <PackageReference Include="Microsoft.Azure.Functions.Worker.Sdk" Version="2.*" />
    <PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.Http" Version="3.*" />
    <PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.Http.AspNetCore" Version="2.*" />
    <PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.DurableTask" Version="1.*" />
    <PackageReference Include="Microsoft.ApplicationInsights.WorkerService" Version="2.*" />
    <PackageReference Include="Microsoft.Azure.Functions.Worker.ApplicationInsights" Version="2.*" />
    <PackageReference Include="Azure.Identity" Version="1.*" />
    <!-- Optional: For class-based orchestrations/activities -->
    <PackageReference Include="Microsoft.DurableTask.Generators" Version="1.*" 
                      OutputItemType="Analyzer" ReferenceOutputAssembly="false" />
  </ItemGroup>
  
  <ItemGroup>
    <None Update="host.json">
      <CopyToOutputDirectory>PreserveNewest</CopyToOutputDirectory>
    </None>
    <None Update="local.settings.json">
      <CopyToOutputDirectory>PreserveNewest</CopyToOutputDirectory>
      <CopyToPublishDirectory>Never</CopyToPublishDirectory>
    </None>
  </ItemGroup>
</Project>
```

### host.json (Durable Task Scheduler)

```json
{
  "version": "2.0",
  "logging": {
    "applicationInsights": {
      "samplingSettings": {
        "isEnabled": true,
        "excludedTypes": "Request"
      },
      "enableLiveMetricsFilters": true
    }
  },
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DTS_CONNECTION_STRING"
      },
      "hubName": "%TASKHUB_NAME%"
    }
  }
}
```

### local.settings.json

```json
{
  "IsEncrypted": false,
  "Values": {
    "FUNCTIONS_WORKER_RUNTIME": "dotnet-isolated",
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "DTS_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
    "TASKHUB_NAME": "default"
  }
}
```

### Program.cs with DI and Logging

```csharp
using Microsoft.Azure.Functions.Worker;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

var host = new HostBuilder()
    .ConfigureFunctionsWorkerDefaults()
    .ConfigureServices(services =>
    {
        // Add Application Insights
        services.AddApplicationInsightsTelemetryWorkerService();
        services.ConfigureFunctionsApplicationInsights();
        
        // Add your services
        services.AddHttpClient();
        services.AddSingleton<IMyService, MyService>();
    })
    .ConfigureLogging(logging =>
    {
        logging.SetMinimumLevel(LogLevel.Information);
    })
    .Build();

await host.RunAsync();
```

## Complete Application Template

### Functions.cs

```csharp
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Entities;
using Microsoft.Extensions.Logging;
using System.Net;

namespace MyDurableFunctions;

public class Functions
{
    private readonly ILogger<Functions> _logger;

    public Functions(ILogger<Functions> logger)
    {
        _logger = logger;
    }

    // HTTP Starter
    [Function("HttpStart")]
    public async Task<HttpResponseData> HttpStart(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/{functionName}")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        string functionName,
        FunctionContext executionContext)
    {
        string? requestBody = await new StreamReader(req.Body).ReadToEndAsync();
        
        var options = new StartOrchestrationOptions
        {
            InstanceId = req.Headers.TryGetValues("X-Instance-Id", out var values) 
                ? values.First() 
                : null
        };
        
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            functionName, requestBody, options);
        
        _logger.LogInformation("Started orchestration {FunctionName} with ID = {InstanceId}", 
            functionName, instanceId);
        
        return await client.CreateCheckStatusResponseAsync(req, instanceId);
    }

    // Get Status
    [Function("GetStatus")]
    public async Task<HttpResponseData> GetStatus(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "orchestrators/{instanceId}/status")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        string instanceId)
    {
        var instance = await client.GetInstanceAsync(instanceId, getInputsAndOutputs: true);
        
        if (instance == null)
        {
            return req.CreateResponse(HttpStatusCode.NotFound);
        }
        
        var response = req.CreateResponse(HttpStatusCode.OK);
        await response.WriteAsJsonAsync(new
        {
            instanceId = instance.InstanceId,
            name = instance.Name,
            runtimeStatus = instance.RuntimeStatus.ToString(),
            createdTime = instance.CreatedAt,
            lastUpdatedTime = instance.LastUpdatedAt,
            input = instance.SerializedInput,
            output = instance.SerializedOutput,
            customStatus = instance.SerializedCustomStatus
        });
        return response;
    }

    // Raise Event
    [Function("RaiseEvent")]
    public async Task<HttpResponseData> RaiseEvent(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/{instanceId}/events/{eventName}")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        string instanceId,
        string eventName)
    {
        string? eventData = await new StreamReader(req.Body).ReadToEndAsync();
        
        await client.RaiseEventAsync(instanceId, eventName, eventData);
        
        return req.CreateResponse(HttpStatusCode.Accepted);
    }

    // Terminate
    [Function("Terminate")]
    public async Task<HttpResponseData> Terminate(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/{instanceId}/terminate")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        string instanceId)
    {
        string? reason = await new StreamReader(req.Body).ReadToEndAsync();
        
        await client.TerminateInstanceAsync(instanceId, reason);
        
        return req.CreateResponse(HttpStatusCode.Accepted);
    }

    // Sample Orchestration
    [Function(nameof(SampleOrchestration))]
    public static async Task<string> SampleOrchestration(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        ILogger logger = context.CreateReplaySafeLogger(nameof(SampleOrchestration));
        string input = context.GetInput<string>() ?? "World";
        
        logger.LogInformation("Starting orchestration with input: {Input}", input);
        
        var result1 = await context.CallActivityAsync<string>(nameof(SayHello), "Tokyo");
        var result2 = await context.CallActivityAsync<string>(nameof(SayHello), "Seattle");
        var result3 = await context.CallActivityAsync<string>(nameof(SayHello), input);
        
        return $"{result1}, {result2}, {result3}";
    }

    // Sample Activity
    [Function(nameof(SayHello))]
    public string SayHello([ActivityTrigger] string name)
    {
        _logger.LogInformation("Saying hello to {Name}", name);
        return $"Hello {name}!";
    }
}
```

## Azure Provisioning

### Azure CLI

```bash
# Variables
RESOURCE_GROUP="my-durable-functions-rg"
LOCATION="eastus"
STORAGE_ACCOUNT="mydurablefuncssa"
FUNCTION_APP="my-durable-functions"
DTS_NAMESPACE="my-dts-namespace"
DTS_SCHEDULER="my-scheduler"
TASKHUB_NAME="default"

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create storage account (required for Azure Functions)
az storage account create \
  --name $STORAGE_ACCOUNT \
  --location $LOCATION \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_LRS

# Create Durable Task Scheduler namespace
az durabletask namespace create \
  --name $DTS_NAMESPACE \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku "Basic"

# Create scheduler within namespace
az durabletask scheduler create \
  --name $DTS_SCHEDULER \
  --namespace-name $DTS_NAMESPACE \
  --resource-group $RESOURCE_GROUP \
  --ip-allow-list "[{\"name\": \"AllowAll\", \"startIPAddress\": \"0.0.0.0\", \"endIPAddress\": \"255.255.255.255\"}]"

# Create task hub
az durabletask taskhub create \
  --name $TASKHUB_NAME \
  --namespace-name $DTS_NAMESPACE \
  --resource-group $RESOURCE_GROUP

# Get scheduler endpoint
DTS_ENDPOINT=$(az durabletask scheduler show \
  --name $DTS_SCHEDULER \
  --namespace-name $DTS_NAMESPACE \
  --resource-group $RESOURCE_GROUP \
  --query "endpoint" -o tsv)

# Create Function App (Consumption plan)
az functionapp create \
  --name $FUNCTION_APP \
  --storage-account $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --consumption-plan-location $LOCATION \
  --runtime dotnet-isolated \
  --runtime-version 8 \
  --functions-version 4 \
  --assign-identity "[system]"

# Get Function App identity
FUNCTION_APP_IDENTITY=$(az functionapp identity show \
  --name $FUNCTION_APP \
  --resource-group $RESOURCE_GROUP \
  --query "principalId" -o tsv)

# Assign Durable Task Contributor role to Function App
DTS_NAMESPACE_ID=$(az durabletask namespace show \
  --name $DTS_NAMESPACE \
  --resource-group $RESOURCE_GROUP \
  --query "id" -o tsv)

az role assignment create \
  --assignee $FUNCTION_APP_IDENTITY \
  --role "Durable Task Data Contributor" \
  --scope $DTS_NAMESPACE_ID

# Configure app settings
az functionapp config appsettings set \
  --name $FUNCTION_APP \
  --resource-group $RESOURCE_GROUP \
  --settings \
    "DTS_CONNECTION_STRING=Endpoint=${DTS_ENDPOINT};Authentication=ManagedIdentity" \
    "TASKHUB_NAME=$TASKHUB_NAME"
```

### Bicep Template

```bicep
// main.bicep
@description('The location for all resources')
param location string = resourceGroup().location

@description('Base name for all resources')
param baseName string = 'mydurablefunc'

// Storage Account
resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: '${baseName}sa'
  location: location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
}

// Durable Task Scheduler Namespace
resource dtsNamespace 'Microsoft.DurableTask/namespaces@2025-11-01' = {
  name: '${baseName}-dts'
  location: location
  sku: {
    name: 'Basic'
    capacity: 1
  }
  properties: {}
}

// Scheduler
resource scheduler 'Microsoft.DurableTask/namespaces/schedulers@2025-11-01' = {
  parent: dtsNamespace
  name: 'scheduler'
  location: location
  properties: {
    ipAllowlist: [
      {
        name: 'AllowAll'
        startIPAddress: '0.0.0.0'
        endIPAddress: '255.255.255.255'
      }
    ]
  }
}

// Task Hub
resource taskHub 'Microsoft.DurableTask/namespaces/taskHubs@2025-11-01' = {
  parent: dtsNamespace
  name: 'default'
  properties: {}
}

// App Service Plan
resource appServicePlan 'Microsoft.Web/serverfarms@2023-01-01' = {
  name: '${baseName}-plan'
  location: location
  sku: {
    name: 'Y1'
    tier: 'Dynamic'
  }
  properties: {}
}

// Function App
resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: '${baseName}-func'
  location: location
  kind: 'functionapp'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    serverFarmId: appServicePlan.id
    siteConfig: {
      netFrameworkVersion: 'v8.0'
      appSettings: [
        {
          name: 'FUNCTIONS_WORKER_RUNTIME'
          value: 'dotnet-isolated'
        }
        {
          name: 'FUNCTIONS_EXTENSION_VERSION'
          value: '~4'
        }
        {
          name: 'AzureWebJobsStorage'
          value: 'DefaultEndpointsProtocol=https;AccountName=${storageAccount.name};EndpointSuffix=${environment().suffixes.storage};AccountKey=${storageAccount.listKeys().keys[0].value}'
        }
        {
          name: 'WEBSITE_CONTENTAZUREFILECONNECTIONSTRING'
          value: 'DefaultEndpointsProtocol=https;AccountName=${storageAccount.name};EndpointSuffix=${environment().suffixes.storage};AccountKey=${storageAccount.listKeys().keys[0].value}'
        }
        {
          name: 'DTS_CONNECTION_STRING'
          value: 'Endpoint=${scheduler.properties.endpoint};Authentication=ManagedIdentity'
        }
        {
          name: 'TASKHUB_NAME'
          value: 'default'
        }
      ]
    }
  }
}

// Role Assignment - Durable Task Data Contributor
resource roleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(dtsNamespace.id, functionApp.id, 'DurableTaskDataContributor')
  scope: dtsNamespace
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '9d3a82f5-2d5a-4c3a-8e7f-6c2f8f8f8f8f') // Durable Task Data Contributor
    principalId: functionApp.identity.principalId
    principalType: 'ServicePrincipal'
  }
}

output functionAppName string = functionApp.name
output functionAppUrl string = 'https://${functionApp.properties.defaultHostName}'
output dtsEndpoint string = scheduler.properties.endpoint
```

```bash
# Deploy
az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file main.bicep \
  --parameters baseName=mydurablefunc
```

## Deployment

### Deploy Function App

```bash
# Build and publish
dotnet publish -c Release -o ./publish

# Create zip
cd publish
zip -r ../deploy.zip .
cd ..

# Deploy to Azure
az functionapp deployment source config-zip \
  --resource-group $RESOURCE_GROUP \
  --name $FUNCTION_APP \
  --src deploy.zip
```

### GitHub Actions Deployment

```yaml
# .github/workflows/deploy.yml
name: Deploy Azure Functions

on:
  push:
    branches: [main]

env:
  AZURE_FUNCTIONAPP_NAME: 'my-durable-functions'
  AZURE_FUNCTIONAPP_PACKAGE_PATH: '.'
  DOTNET_VERSION: '8.0.x'

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout
      uses: actions/checkout@v4

    - name: Setup .NET
      uses: actions/setup-dotnet@v4
      with:
        dotnet-version: ${{ env.DOTNET_VERSION }}

    - name: Build
      run: dotnet build --configuration Release

    - name: Publish
      run: dotnet publish -c Release -o ${{ env.AZURE_FUNCTIONAPP_PACKAGE_PATH }}/output

    - name: Deploy to Azure Functions
      uses: Azure/functions-action@v1
      with:
        app-name: ${{ env.AZURE_FUNCTIONAPP_NAME }}
        package: '${{ env.AZURE_FUNCTIONAPP_PACKAGE_PATH }}/output'
        publish-profile: ${{ secrets.AZURE_FUNCTIONAPP_PUBLISH_PROFILE }}
```

## Deployment Options

### Azure Container Apps

```bash
# Create environment
az containerapp env create \
  --name $ENVIRONMENT_NAME \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION

# Build and push image
az acr build \
  --registry $ACR_NAME \
  --image $IMAGE_NAME:$TAG .

# Deploy
az containerapp create \
  --name $FUNCTION_APP \
  --resource-group $RESOURCE_GROUP \
  --environment $ENVIRONMENT_NAME \
  --image $ACR_NAME.azurecr.io/$IMAGE_NAME:$TAG \
  --target-port 80 \
  --ingress 'external' \
  --min-replicas 1 \
  --max-replicas 10 \
  --env-vars \
    "FUNCTIONS_WORKER_RUNTIME=dotnet-isolated" \
    "AzureWebJobsStorage=$STORAGE_CONNECTION_STRING" \
    "DTS_CONNECTION_STRING=Endpoint=${DTS_ENDPOINT};Authentication=ManagedIdentity" \
    "TASKHUB_NAME=$TASKHUB_NAME" \
  --user-assigned $MANAGED_IDENTITY_ID
```

### Dockerfile

```dockerfile
FROM mcr.microsoft.com/azure-functions/dotnet-isolated:4-dotnet-isolated8.0 AS base
WORKDIR /home/site/wwwroot
EXPOSE 80

FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY ["MyDurableFunctions.csproj", "."]
RUN dotnet restore
COPY . .
RUN dotnet build -c Release -o /app/build

FROM build AS publish
RUN dotnet publish -c Release -o /app/publish

FROM base AS final
WORKDIR /home/site/wwwroot
COPY --from=publish /app/publish .
ENV AzureWebJobsScriptRoot=/home/site/wwwroot
ENV AzureFunctionsJobHost__Logging__Console__IsEnabled=true
```

## Testing

### Unit Testing with Moq

```csharp
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Moq;
using Xunit;

public class OrchestratorTests
{
    [Fact]
    public async Task SampleOrchestration_ReturnsExpectedResult()
    {
        // Arrange
        var contextMock = new Mock<TaskOrchestrationContext>();
        
        contextMock.Setup(x => x.GetInput<string>()).Returns("Test");
        
        contextMock
            .Setup(x => x.CallActivityAsync<string>(nameof(Functions.SayHello), "Tokyo", It.IsAny<TaskOptions>()))
            .ReturnsAsync("Hello Tokyo!");
        
        contextMock
            .Setup(x => x.CallActivityAsync<string>(nameof(Functions.SayHello), "Seattle", It.IsAny<TaskOptions>()))
            .ReturnsAsync("Hello Seattle!");
        
        contextMock
            .Setup(x => x.CallActivityAsync<string>(nameof(Functions.SayHello), "Test", It.IsAny<TaskOptions>()))
            .ReturnsAsync("Hello Test!");
        
        contextMock
            .Setup(x => x.CreateReplaySafeLogger(It.IsAny<string>()))
            .Returns(Mock.Of<ILogger>());

        // Act
        var result = await Functions.SampleOrchestration(contextMock.Object);

        // Assert
        Assert.Equal("Hello Tokyo!, Hello Seattle!, Hello Test!", result);
    }
}
```

### Integration Testing

```csharp
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Xunit;

public class IntegrationTests : IAsyncLifetime
{
    private IHost _host = null!;
    private DurableTaskClient _client = null!;

    public async Task InitializeAsync()
    {
        _host = new HostBuilder()
            .ConfigureFunctionsWorkerDefaults()
            .Build();
        
        await _host.StartAsync();
        
        _client = _host.Services.GetRequiredService<DurableTaskClient>();
    }

    public async Task DisposeAsync()
    {
        await _host.StopAsync();
        _host.Dispose();
    }

    [Fact]
    public async Task Orchestration_Completes_Successfully()
    {
        // Schedule orchestration
        string instanceId = await _client.ScheduleNewOrchestrationInstanceAsync(
            nameof(Functions.SampleOrchestration), "IntegrationTest");
        
        // Wait for completion
        var result = await _client.WaitForInstanceCompletionAsync(
            instanceId, 
            getInputsAndOutputs: true,
            cancellationToken: new CancellationTokenSource(TimeSpan.FromSeconds(30)).Token);
        
        Assert.Equal(OrchestrationRuntimeStatus.Completed, result!.RuntimeStatus);
    }
}
```

## Monitoring and Logging

### Application Insights

```csharp
// Program.cs
using Microsoft.ApplicationInsights.Extensibility;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

var host = new HostBuilder()
    .ConfigureFunctionsWorkerDefaults()
    .ConfigureServices(services =>
    {
        services.AddApplicationInsightsTelemetryWorkerService();
        services.ConfigureFunctionsApplicationInsights();
        
        // Custom telemetry processor
        services.AddSingleton<ITelemetryInitializer, CustomTelemetryInitializer>();
    })
    .Build();

await host.RunAsync();

public class CustomTelemetryInitializer : ITelemetryInitializer
{
    public void Initialize(ITelemetry telemetry)
    {
        telemetry.Context.Cloud.RoleName = "MyDurableFunctions";
    }
}
```

### Custom Status and Metrics

```csharp
[Function(nameof(TrackedOrchestration))]
public static async Task<string> TrackedOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    ILogger logger = context.CreateReplaySafeLogger(nameof(TrackedOrchestration));
    
    // Set progress status
    context.SetCustomStatus(new { Stage = "Starting", Progress = 0 });
    
    await context.CallActivityAsync<string>(nameof(Step1), null);
    context.SetCustomStatus(new { Stage = "Step1Complete", Progress = 33 });
    
    await context.CallActivityAsync<string>(nameof(Step2), null);
    context.SetCustomStatus(new { Stage = "Step2Complete", Progress = 66 });
    
    await context.CallActivityAsync<string>(nameof(Step3), null);
    context.SetCustomStatus(new { Stage = "Completed", Progress = 100 });
    
    return "Done";
}
```

### KQL Queries for Monitoring

```kql
// Orchestration completion times
traces
| where message contains "orchestration"
| summarize avg(duration) by bin(timestamp, 1h)

// Failed orchestrations
traces
| where severityLevel >= 3
| where message contains "orchestration" or message contains "activity"
| project timestamp, message, severityLevel

// Activity execution times
customMetrics
| where name == "ActivityDuration"
| summarize avg(value), percentile(value, 95) by bin(timestamp, 5m)
```
