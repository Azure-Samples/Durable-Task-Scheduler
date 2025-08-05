# Agent Chaining Sample with Durable Task SDK for .NET

This sample demonstrates an AI-powered news article generator using the Durable Task .NET SDK. The workflow chains multiple specialized AI agents to research topics, generate content, create images, and produce an HTML article.

## What This Sample Does

This application chains together three AI agents to create a complete news article:

1. **Research Agent**: Researches a topic using web search capabilities
2. **Content Generation Agent**: Creates article content based on research findings
3. **Image Generation Agent**: Creates images for the article using DALL-E

The workflow is orchestrated using the Durable Task SDK, which handles the workflow coordination and reliability. When complete, the article is saved as an HTML file with text content and images.

![Agent Chaining Architecture](images/architecture.png)

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Docker](https://www.docker.com/products/docker-desktop/) (for the Durable Task Scheduler emulator)
- Azure AI Projects service
- Azure credentials (via az login)
- Optional: Azure OpenAI service with DALL-E 3

## Running the Sample Locally

### Quick Start Guide

1. **Start the Durable Task Scheduler Emulator**

   ```bash
   # Pull and run the emulator
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. **Set Environment Variables**

   ```bash
   # Required: Azure AI Projects endpoint
   export AGENT_CONNECTION_STRING="https://{region}.aiprojects.azure.com/api/projects/{resourceGroup}/{projectName}"
   
   # Note: The AGENT_CONNECTION_STRING must be in URL format as shown above.
   # The format should match: https://{region}.aiprojects.azure.com/api/projects/{resourceGroup}/{projectName}
   # Example: https://eastus.aiprojects.azure.com/api/projects/my-resource-group/my-ai-project

   # Optional: OpenAI model name (defaults to "gpt-4")
   export OPENAI_DEPLOYMENT_NAME="gpt-4-turbo"

   # Optional: For image generation (if omitted, placeholder images will be used)
   export DALLE_ENDPOINT="https://your-resource.openai.azure.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01"
   ```

   **Note for Windows users:**
   - Command Prompt: Use `set` instead of `export`
   - PowerShell: Use `$env:VARIABLE_NAME="value"`

3. **Build and Run the Application**

   In the first terminal:
   ```bash
   dotnet build
   dotnet run --project Worker/Worker.csproj
   ```

   In a second terminal:
   ```bash
   dotnet run --project Client/Client.csproj
   ```

4. **Generate an Article**

   **Option 1: Using the HTTP API**
   
   The client exposes an HTTP API endpoint at http://localhost:5000. You can use tools like curl or any HTTP client to interact with it:

   ```bash
   # Make a request to generate an article
   curl -X POST http://localhost:5000/api/articles \
     -H "Content-Type: application/json" \
     -d '{"topic": "renewable energy innovations"}'
   
   # Check the status of an article generation
   curl http://localhost:5000/api/articles/{instanceId}
   ```
   
   There is also a test.http file that can be used with the VS Code REST Client extension.
   
   **Option 2: Using the Console Interface**

   Alternatively, follow the prompts in the client console to enter a news topic. The application will:
   - Research the topic
   - Generate article content
   - Create supporting images 
   - Save an HTML file with the complete article

   When finished, the client will show the path to your generated HTML file (typically in a temp directory like `/var/folders/.../T/article-generator/` on macOS).

## Deploy to Azure

This sample includes everything needed to deploy to Azure using the Azure Developer CLI (azd). The deployment will automatically provision all required Azure resources and deploy your application.

### Prerequisites

Before deploying to Azure, ensure you have:

- **Azure CLI**: [Download and install](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
- **Azure Developer CLI (azd)**: [Download and install](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
- **Azure Subscription**: You'll need an active Azure subscription
- **Docker**: Required for building container images

### Step-by-Step Deployment

1. **Login to Azure**
   ```bash
   # Login with your Azure account
   az login
   
   # Set your subscription (if you have multiple)
   az account set --subscription "your-subscription-id"
   ```

2. **Initialize the Azure Developer CLI**
   ```bash
   # Initialize azd in the project directory
   azd init
   
   # This will detect the existing azure.yaml configuration
   ```

3. **Deploy to Azure**
   ```bash
   # Deploy the entire application with one command
   azd up
   ```

   The `azd up` command will:
   - **Provision Infrastructure**: Create all required Azure resources
   - **Build Images**: Build Docker containers for the client and worker
   - **Deploy Services**: Deploy to Azure Container Apps
   - **Configure Networking**: Set up ingress and internal communication
   - **Set Environment Variables**: Configure all necessary settings

### What Gets Deployed

The deployment creates the following Azure resources:

| Resource | Type | Purpose |
|----------|------|---------|
| **Resource Group** | `Microsoft.Resources/resourceGroups` | Contains all project resources |
| **Container Apps Environment** | `Microsoft.App/managedEnvironments` | Runtime environment for containers |
| **Container Registry** | `Microsoft.ContainerRegistry/registries` | Stores Docker images |
| **Client App** | `Microsoft.App/containerApps` | Web API frontend (publicly accessible) |
| **Worker App** | `Microsoft.App/containerApps` | Background worker (internal only) |
| **Durable Task Scheduler** | `Microsoft.DurableTask/schedulers` | Orchestration engine |
| **AI Project** | `Microsoft.CognitiveServices/accounts` | Azure AI services |
| **OpenAI Deployments** | `Microsoft.CognitiveServices/accounts/deployments` | GPT-4o-mini and DALL-E 3 models |
| **Log Analytics** | `Microsoft.OperationalInsights/workspaces` | Application monitoring and logs |
| **Managed Identity** | `Microsoft.ManagedIdentity/userAssignedIdentities` | Secure authentication |

### Post-Deployment

After successful deployment, you'll see output similar to:

```bash
SUCCESS: Your application was deployed to Azure in 4 minutes.

You can view the resources created under the resource group rg-<environment-name> in Azure Portal:
https://portal.azure.com/#@/resource/subscriptions/.../resourceGroups/rg-<environment-name>/overview

Services:
  client    https://ca-<unique-id>-client.<region>.azurecontainerapps.io/
  worker    (internal only)
```

### Using Your Deployed Application

1. **Test the API**
   
   Use the provided client URL to interact with your deployed application:
   
   ```bash
   # Replace <client-url> with your actual deployment URL
   curl -X POST <client-url>/api/content \
     -H "Content-Type: application/json" \
     -d '{"topic": "Latest developments in AI technology"}'
   ```

2. **View Generated Articles**
   
   When an orchestration completes, the response will include an `articleEndpoint` property:
   
   ```json
   {
     "instanceId": "abc123...",
     "articleEndpoint": "https://ca-xyz-client.region.azurecontainerapps.io/api/content/abc123.../document",
     "status": "Completed"
   }
   ```
   
   Open the `articleEndpoint` URL in your browser to view the generated HTML article.

3. **Monitor Your Application**
   
   - **Azure Portal**: View resources and metrics
   - **Container Apps Logs**: Monitor application logs and performance
   - **Log Analytics**: Query detailed telemetry and traces

### Managing Your Deployment

**Update your application:**
```bash
# Deploy code changes
azd deploy

# Update infrastructure and deploy
azd up
```

**View deployment status:**
```bash
# Show current deployment information
azd show

# View environment variables
azd env get-values
```

**Clean up resources:**
```bash
# Remove all Azure resources (be careful!)
azd down
```

### Troubleshooting Deployment

**Common Issues:**

1. **Insufficient permissions**: Ensure your Azure account has `Contributor` or `Owner` role
2. **Resource naming conflicts**: Try a different environment name with `azd env set AZURE_ENV_NAME <new-name>`
3. **Deployment timeout**: Container builds can take time; wait for completion
4. **Region availability**: Some Azure AI services may not be available in all regions

**View detailed logs:**
```bash
# View container app logs
az containerapp logs show --name <app-name> --resource-group <rg-name>

# View deployment history
azd deploy --debug
```

### Cost Considerations

The deployed resources will incur costs based on usage:
- **Container Apps**: Pay-per-use scaling model
- **Azure OpenAI**: Token-based pricing for GPT and DALL-E
- **Container Registry**: Storage costs for images
- **Log Analytics**: Data ingestion and retention costs

Consider setting up [Azure Cost Management](https://learn.microsoft.com/en-us/azure/cost-management-billing/) alerts to monitor spending.
## How It Works

The application uses a durable orchestration workflow with four key steps:

1. **Research**: The Research Agent gathers facts, sources, and angles for your topic
2. **Content Creation**: The Content Generation Agent writes a professional news article
3. **Image Generation**: The Image Generation Agent creates descriptive images with DALL-E
4. **Assembly**: All content is combined into a styled HTML document

![sample-article](images/sample-article.png)

## Authentication

The sample uses Azure's DefaultAzureCredential for authentication:

```bash
# Login with your Azure account
az login

# Select the correct subscription if needed
az account set --subscription "your-subscription-name-or-id"
```

Ensure your account has:
- "AI Project User" role for Azure AI Projects
- "Cognitive Services OpenAI User" role for Azure OpenAI (if using DALL-E)

## View the Orchestration Dashboard

Monitor your workflow executions in the Durable Task Scheduler dashboard:
1. Open a browser and go to `http://localhost:8082`
2. Select the `default` task hub
3. Check the Instances tab to see your orchestrations

## Learn More

- [Durable Task SDK for .NET](https://github.com/microsoft/durabletask-dotnet)
- [Azure AI Projects Documentation](https://learn.microsoft.com/azure/ai-services/ai-project/overview)
- [Azure OpenAI Service](https://learn.microsoft.com/azure/ai-services/openai/)
