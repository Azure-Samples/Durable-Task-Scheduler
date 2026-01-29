# Azure Durable Task Scheduler Setup

## Table of Contents
- [Local Development with Emulator](#local-development-with-emulator)
- [Azure Deployment](#azure-deployment)
- [Authentication Configuration](#authentication-configuration)
- [ASP.NET Integration](#aspnet-integration)
- [Worker Console App](#worker-console-app)
- [Deployment Options](#deployment-options)

---

## Local Development with Emulator

### Docker Setup

```bash
# Pull the emulator image
docker pull mcr.microsoft.com/dts/dts-emulator:latest

# Run the emulator
docker run -d \
  --name dts-emulator \
  -p 8080:8080 \
  -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest

# Ports:
# - 8080: gRPC endpoint for worker/client connections
# - 8082: Dashboard UI

# View logs
docker logs -f dts-emulator

# Stop emulator
docker stop dts-emulator

# Remove container
docker rm dts-emulator
```

### Emulator Connection String

```csharp
var connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
```

### Dashboard Access

Navigate to `http://localhost:8082` to view:
- Orchestration instances
- Execution history
- Entity states
- Activity progress

---

## Azure Deployment

### Prerequisites

```bash
# Install/upgrade Azure CLI
az upgrade

# Install Durable Task CLI extension
az extension add --name durabletask --allow-preview true

# Login to Azure
az login
```

### Create Resources

```bash
# Set variables
RESOURCE_GROUP="my-dts-rg"
LOCATION="eastus"  # Check available regions first
SCHEDULER_NAME="my-dts-scheduler"
TASKHUB_NAME="my-taskhub"

# Check available regions
az provider show \
  --namespace Microsoft.DurableTask \
  --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" \
  --out table

# Create resource group
az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION

# Create scheduler
az durabletask scheduler create \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --ip-allowlist '["0.0.0.0/0"]' \
  --sku-name "Dedicated" \
  --sku-capacity 1 \
  --tags "environment=development"

# Create task hub
az durabletask taskhub create \
  --resource-group $RESOURCE_GROUP \
  --scheduler-name $SCHEDULER_NAME \
  --name $TASKHUB_NAME
```

### Grant Access

```bash
# Get subscription and user info
SUBSCRIPTION_ID=$(az account show --query "id" -o tsv)
USER_EMAIL=$(az account show --query "user.name" -o tsv)

# Grant "Durable Task Data Contributor" role
az role assignment create \
  --assignee $USER_EMAIL \
  --role "Durable Task Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.DurableTask/schedulers/$SCHEDULER_NAME/taskHubs/$TASKHUB_NAME"

# Note: Role assignment may take 1-2 minutes to propagate
```

### Get Connection Details

```bash
# Get scheduler endpoint
ENDPOINT=$(az durabletask scheduler show \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --query "properties.endpoint" \
  --output tsv)

# Get dashboard URL
DASHBOARD_URL=$(az durabletask taskhub show \
  --resource-group $RESOURCE_GROUP \
  --scheduler-name $SCHEDULER_NAME \
  --name $TASKHUB_NAME \
  --query "properties.dashboardUrl" \
  --output tsv)

# Set environment variable
export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=$ENDPOINT;TaskHub=$TASKHUB_NAME;Authentication=DefaultAzure"

echo "Endpoint: $ENDPOINT"
echo "Dashboard: $DASHBOARD_URL"
echo "Connection String: $DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
```

---

## Authentication Configuration

### Authentication Types

| Type | Use Case | Connection String Value |
|------|----------|------------------------|
| None | Local emulator only | `Authentication=None` |
| DefaultAzure | Auto-detect credential (recommended) | `Authentication=DefaultAzure` |
| ManagedIdentity | Azure hosted apps with MI | `Authentication=ManagedIdentity` |
| WorkloadIdentity | Kubernetes with workload identity | `Authentication=WorkloadIdentity` |
| AzureCLI | Local dev with az login | `Authentication=AzureCLI` |
| Environment | Service principal via env vars | `Authentication=Environment` |

### DefaultAzureCredential Chain

The `DefaultAzure` authentication tries these methods in order:
1. Environment variables (service principal)
2. Workload Identity
3. Managed Identity
4. Visual Studio credential
5. Azure CLI credential
6. Azure PowerShell credential
7. Interactive browser (development only)

### Service Principal (Environment Variables)

```bash
# Required environment variables for Environment auth
export AZURE_CLIENT_ID="<application-id>"
export AZURE_TENANT_ID="<tenant-id>"
export AZURE_CLIENT_SECRET="<client-secret>"
```

### Managed Identity Setup

```bash
# Create user-assigned managed identity
az identity create \
  --name "dts-app-identity" \
  --resource-group $RESOURCE_GROUP

# Get identity's principal ID
IDENTITY_PRINCIPAL_ID=$(az identity show \
  --name "dts-app-identity" \
  --resource-group $RESOURCE_GROUP \
  --query "principalId" -o tsv)

# Grant role to managed identity
az role assignment create \
  --assignee $IDENTITY_PRINCIPAL_ID \
  --role "Durable Task Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.DurableTask/schedulers/$SCHEDULER_NAME/taskHubs/$TASKHUB_NAME"
```

---

## ASP.NET Integration

### Project Setup

```xml
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Microsoft.DurableTask.Client.AzureManaged" Version="1.*" />
    <PackageReference Include="Microsoft.DurableTask.Worker.AzureManaged" Version="1.*" />
    <PackageReference Include="Microsoft.DurableTask.Generators" Version="1.*" OutputItemType="Analyzer" />
    <PackageReference Include="Azure.Identity" Version="1.*" />
  </ItemGroup>
</Project>
```

### Program.cs

```csharp
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Worker;

var builder = WebApplication.CreateBuilder(args);

// Get connection string from configuration
var connectionString = builder.Configuration["DurableTaskScheduler:ConnectionString"]
    ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

// Add Durable Task Worker (background service)
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddAllGeneratedTasks();
    })
    .UseDurableTaskScheduler(connectionString);

// Add Durable Task Client (for scheduling orchestrations)
builder.Services.AddDurableTaskClient()
    .UseDurableTaskScheduler(connectionString);

builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

var app = builder.Build();

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseAuthorization();
app.MapControllers();

app.Run();
```

### Controller Example

```csharp
using Microsoft.AspNetCore.Mvc;
using Microsoft.DurableTask.Client;

[ApiController]
[Route("api/[controller]")]
public class WorkflowsController : ControllerBase
{
    private readonly DurableTaskClient _client;

    public WorkflowsController(DurableTaskClient client)
    {
        _client = client;
    }

    [HttpPost("start")]
    public async Task<IActionResult> StartWorkflow([FromBody] WorkflowRequest request)
    {
        var instanceId = await _client.ScheduleNewOrchestrationInstanceAsync(
            "MyOrchestration",
            request.Input);
        
        return Accepted(new { InstanceId = instanceId });
    }

    [HttpGet("{instanceId}/status")]
    public async Task<IActionResult> GetStatus(string instanceId)
    {
        var metadata = await _client.GetInstanceAsync(instanceId, getInputsAndOutputs: true);
        
        if (metadata == null)
            return NotFound();
        
        return Ok(new
        {
            InstanceId = metadata.InstanceId,
            Status = metadata.RuntimeStatus.ToString(),
            CreatedAt = metadata.CreatedAt,
            CompletedAt = metadata.LastUpdatedAt,
            Output = metadata.SerializedOutput
        });
    }

    [HttpPost("{instanceId}/events/{eventName}")]
    public async Task<IActionResult> RaiseEvent(
        string instanceId, 
        string eventName, 
        [FromBody] object eventData)
    {
        await _client.RaiseEventAsync(instanceId, eventName, eventData);
        return Accepted();
    }

    [HttpPost("{instanceId}/terminate")]
    public async Task<IActionResult> Terminate(string instanceId, [FromBody] string reason)
    {
        await _client.TerminateInstanceAsync(instanceId, reason);
        return Accepted();
    }
}
```

### appsettings.json

```json
{
  "DurableTaskScheduler": {
    "ConnectionString": "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
  },
  "Logging": {
    "LogLevel": {
      "Default": "Information",
      "Microsoft.DurableTask": "Debug"
    }
  }
}
```

### appsettings.Production.json

```json
{
  "DurableTaskScheduler": {
    "ConnectionString": "Endpoint=https://my-scheduler.eastus.durabletask.io;TaskHub=production;Authentication=ManagedIdentity"
  }
}
```

---

## Worker Console App

### Project Setup

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Microsoft.DurableTask.Worker.AzureManaged" Version="1.*" />
    <PackageReference Include="Microsoft.DurableTask.Generators" Version="1.*" OutputItemType="Analyzer" />
    <PackageReference Include="Microsoft.Extensions.Hosting" Version="8.*" />
    <PackageReference Include="Azure.Identity" Version="1.*" />
  </ItemGroup>
</Project>
```

### Program.cs (Worker Only)

```csharp
using Microsoft.DurableTask.Worker;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

var builder = Host.CreateApplicationBuilder(args);

// Configure logging
builder.Logging.AddConsole();
builder.Logging.SetMinimumLevel(LogLevel.Information);

var connectionString = GetConnectionString();

builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddAllGeneratedTasks();
    })
    .UseDurableTaskScheduler(connectionString);

var host = builder.Build();

Console.WriteLine("Starting Durable Task Worker...");
Console.WriteLine($"Connection: {MaskConnectionString(connectionString)}");

await host.RunAsync();

static string GetConnectionString()
{
    var endpoint = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
    var taskHub = Environment.GetEnvironmentVariable("TASKHUB") ?? "default";
    var authType = endpoint.StartsWith("http://localhost") ? "None" : "DefaultAzure";
    
    return $"Endpoint={endpoint};TaskHub={taskHub};Authentication={authType}";
}

static string MaskConnectionString(string connStr)
{
    // Don't log full connection string in production
    if (connStr.Contains("localhost"))
        return connStr;
    return connStr.Split(';')[0] + ";...";
}
```

---

## Deployment Options

### Azure Container Apps

```bash
# Create Container Apps environment
az containerapp env create \
  --name dts-env \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION

# Deploy worker app
az containerapp create \
  --name dts-worker \
  --resource-group $RESOURCE_GROUP \
  --environment dts-env \
  --image myregistry.azurecr.io/dts-worker:latest \
  --min-replicas 1 \
  --max-replicas 5 \
  --cpu 0.5 \
  --memory 1.0Gi \
  --user-assigned /subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/dts-app-identity \
  --env-vars \
    ENDPOINT=$ENDPOINT \
    TASKHUB=$TASKHUB_NAME
```

### Azure Kubernetes Service (AKS)

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dts-worker
spec:
  replicas: 2
  selector:
    matchLabels:
      app: dts-worker
  template:
    metadata:
      labels:
        app: dts-worker
        azure.workload.identity/use: "true"
    spec:
      serviceAccountName: dts-service-account
      containers:
      - name: worker
        image: myregistry.azurecr.io/dts-worker:latest
        env:
        - name: ENDPOINT
          valueFrom:
            secretKeyRef:
              name: dts-config
              key: endpoint
        - name: TASKHUB
          valueFrom:
            secretKeyRef:
              name: dts-config
              key: taskhub
        resources:
          requests:
            cpu: "250m"
            memory: "512Mi"
          limits:
            cpu: "1000m"
            memory: "1Gi"
```

### Azure App Service

```bash
# Create App Service Plan
az appservice plan create \
  --name dts-plan \
  --resource-group $RESOURCE_GROUP \
  --sku B1 \
  --is-linux

# Create Web App
az webapp create \
  --name dts-webapp \
  --resource-group $RESOURCE_GROUP \
  --plan dts-plan \
  --runtime "DOTNET|8.0"

# Assign managed identity
az webapp identity assign \
  --name dts-webapp \
  --resource-group $RESOURCE_GROUP

# Configure app settings
az webapp config appsettings set \
  --name dts-webapp \
  --resource-group $RESOURCE_GROUP \
  --settings \
    DurableTaskScheduler__ConnectionString="Endpoint=$ENDPOINT;TaskHub=$TASKHUB_NAME;Authentication=ManagedIdentity"
```

### Docker Compose (Development)

```yaml
# docker-compose.yml
version: '3.8'

services:
  emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"
      - "8082:8082"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  worker:
    build: ./Worker
    depends_on:
      emulator:
        condition: service_healthy
    environment:
      - ENDPOINT=http://emulator:8080
      - TASKHUB=default

  api:
    build: ./Api
    ports:
      - "5000:8080"
    depends_on:
      emulator:
        condition: service_healthy
    environment:
      - DurableTaskScheduler__ConnectionString=Endpoint=http://emulator:8080;TaskHub=default;Authentication=None
```
