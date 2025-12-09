# AI Travel Planner with Durable Agents

A travel planning application that demonstrates how to build **durable AI agents** using the [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/user-guide/agents/agent-types/durable-agent/create-durable-agent). The application coordinates multiple specialized AI agents to create comprehensive, personalized travel plans through a structured workflow.

## Overview

This sample showcases an **agentic workflow** where specialized AI agents collaborate to plan travel experiences. Each agent focuses on a specific aspect of travel planning—destination recommendations, itinerary creation, and local insights—orchestrated by the Durable Task extension for reliability and state management.

### Why Durable Agents?

Traditional AI agents can be unpredictable and inconsistent. The [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/user-guide/agents/agent-types/durable-agent/create-durable-agent) solves this by providing:

- **Deterministic workflows**: Pre-defined steps ensure consistent, high-quality results
- **Built-in resilience**: Automatic state persistence and recovery from failures  
- **Human-in-the-loop**: Native support for approval workflows before booking
- **Scalability**: Serverless execution that scales with demand

## Architecture

![architecture](./images/architecture.png)

### Workflow

1. **User Request** → User submits travel preferences via React frontend
2. **Destination Recommendation** → AI agent analyzes preferences and suggests destinations
3. **Itinerary Planning** → AI agent creates detailed day-by-day plans
4. **Local Recommendations** → AI agent adds insider tips and attractions
5. **Storage** → Travel plan saved to Azure Blob Storage
6. **Approval** → User reviews and approves the plan
7. **Booking** → Upon approval, booking process completes

### Tech Stack

| Component | Technology |
|-----------|------------|
| **Backend** | .NET 9, Azure Functions (Isolated Worker) |
| **AI Framework** | Microsoft Agent Framework with Durable Task Extension |
| **Orchestration** | Durable Task Scheduler |
| **AI Model** | Azure OpenAI (GPT-4o-mini) |
| **Frontend** | React |
| **Hosting** | Azure Static Web Apps, Azure Functions |
| **Storage** | Azure Blob Storage |
| **Infrastructure** | Bicep, Azure Developer CLI (azd) |

## Prerequisites

Before you begin, ensure you have the following installed:

- [.NET 9 SDK](https://dotnet.microsoft.com/download/dotnet/9.0)
- [Node.js 18+](https://nodejs.org/) and npm
- [Azure Developer CLI (azd)](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
- [Azure Functions Core Tools v4](https://docs.microsoft.com/en-us/azure/azure-functions/functions-run-local)
- [Docker](https://www.docker.com/get-started) (for local development)
- An Azure subscription with permissions to create resources

## Deploy to Azure

### 1. Clone the Repository

**Bash**
```bash
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler/samples/durable-functions/dotnet/AiAgentTravelPlanOrchestrator
```

**PowerShell**
```powershell
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler\samples\durable-functions\dotnet\AiAgentTravelPlanOrchestrator
```

### 2. Login to Azure

**Bash**
```bash
azd auth login
az login
```

**PowerShell**
```powershell
azd auth login
az login
```

### 3. Provision and Deploy

Run the following command to provision all Azure resources and deploy the application:

**Bash**
```bash
azd up
```

**PowerShell**
```powershell
azd up
```

This command will:
- Create a new resource group
- Provision Azure OpenAI, Durable Task Scheduler, Storage, Functions, and Static Web App
- Build and deploy the backend API
- Build and deploy the frontend React application

Follow the prompts to select your subscription and region.

### 4. Deploy the Web Frontend

After the initial deployment, deploy the web frontend with the correct API URL:

**Bash**
```bash
azd package web
azd deploy web
```

**PowerShell**
```powershell
azd package web
azd deploy web
```

### 5. Access the Application

Once deployment completes, the CLI will output the URLs for your services:

- **Frontend**: `https://<your-static-web-app>.azurestaticapps.net`
- **API**: `https://<your-function-app>.azurewebsites.net`

## Local Development

### 1. Start Azure Storage Emulator

**Bash**
```bash
npm install -g azurite
azurite --silent --location ./azurite &
```

**PowerShell**
```powershell
npm install -g azurite
Start-Process azurite -ArgumentList "--silent", "--location", "./azurite"
```

### 2. Start Durable Task Scheduler Emulator

**Bash**
```bash
docker run -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:latest
```

**PowerShell**
```powershell
docker run -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:latest
```

### 3. Configure Local Settings

Create a `local.settings.json` file:

```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "dotnet-isolated",
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
    "TASKHUB_NAME": "default",
    "AZURE_OPENAI_ENDPOINT": "https://<your-endpoint>.openai.azure.com/",
    "AZURE_OPENAI_DEPLOYMENT_NAME": "gpt-4o-mini"
  },
  "Host": {
    "LocalHttpPort": 7071,
    "CORS": "*"
  }
}
```

> **Note**: The application uses `DefaultAzureCredential` for authentication. Run `az login` before starting the application.

### 4. Start the Backend

**Bash**
```bash
func start
```

**PowerShell**
```powershell
func start
```

### 5. Start the Frontend

**Bash**
```bash
cd Frontend
npm install
npm start
```

**PowerShell**
```powershell
cd Frontend
npm install
npm start
```

The application will be available at `http://localhost:3000`.

## Clean Up

To remove all Azure resources and avoid ongoing charges:

**Bash**
```bash
azd down --purge
```

**PowerShell**
```powershell
azd down --purge
```

## Learn More

- [Durable Task Extension for Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/user-guide/agents/agent-types/durable-agent/create-durable-agent)
- [Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/overview/agent-framework-overview)
- [Azure Durable Task Scheduler](https://learn.microsoft.com/en-us/azure/azure-functions/durable/durable-task-scheduler/quickstart-durable-task-scheduler)
- [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
