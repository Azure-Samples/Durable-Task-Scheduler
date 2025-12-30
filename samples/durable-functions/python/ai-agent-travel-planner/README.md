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
| **Backend** | Python 3.11, Azure Functions (Flex Consumption) |
| **AI Framework** | Microsoft Agent Framework with Durable Task Extension |
| **Orchestration** | Durable Task Scheduler |
| **AI Model** | Azure OpenAI (GPT-4o-mini) |
| **Frontend** | React |
| **Hosting** | Azure Static Web Apps, Azure Functions |
| **Storage** | Azure Blob Storage |
| **Infrastructure** | Bicep, Azure Developer CLI (azd) |

## Prerequisites

Before you begin, ensure you have the following installed:

- [Python 3.11](https://www.python.org/downloads/)
- [Node.js 18+](https://nodejs.org/) and npm
- [Azure Developer CLI (azd)](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
- [Azure Functions Core Tools v4](https://docs.microsoft.com/en-us/azure/azure-functions/functions-run-local)
- [Docker](https://www.docker.com/get-started) (for local development)
- An Azure subscription with permissions to create resources

## Deploy to Azure

### 1. Clone the Repository

```bash
git clone https://github.com/Azure-Samples/Durable-Task-Scheduler.git
cd Durable-Task-Scheduler/samples/durable-functions/python/ai-agent-travel-planner
```

### 2. Login to Azure

```bash
azd auth login
az login
```

### 3. Choose a Region

When selecting a region for deployment, you need to ensure it supports the **GPT-4o-mini** model. Use the Azure CLI to verify that your chosen region has quota available for the GPT-4o-mini model:

**Bash**
```bash
az cognitiveservices usage list --location "westus3" | \
    jq -r '.[] | select(.name.value | contains("OpenAI.Standard.gpt-4o-mini"))
               | select(.currentValue < .limit)
               | .name.value'
```

**PowerShell**
```powershell
az cognitiveservices usage list --location "westus3" | 
    ConvertFrom-Json | 
    Where-Object { $_.name.value -like "*OpenAI.Standard.gpt-4o-mini*" -and $_.currentValue -lt $_.limit } | 
    Select-Object -ExpandProperty name | 
    Select-Object -ExpandProperty value
```

Replace `"westus3"` with your desired region. If the model is available and you have quota, it will appear in the output.

### 4. Set Model Location

Set the `MODEL_LOCATION` environment variable to the region where you have Cognitive Services access:

**Bash**
```bash
export MODEL_LOCATION="westus3"
```

**PowerShell**
```powershell
$env:MODEL_LOCATION="westus3"
```

Replace `"westus3"` with your chosen region from step 3.

### 5. Provision and Deploy

Run the following command to provision all Azure resources and deploy the application:

```bash
azd up
```

This command will:

- Create a new resource group
- Provision Azure OpenAI, Durable Task Scheduler, Storage, Functions, and Static Web App
- Build and deploy the backend API
- Build and deploy the frontend React application

Follow the prompts to select your subscription and region.

### 6. Access the Application

Once deployment completes, the CLI will output the URLs for your services:

- **Frontend**: `https://<your-static-web-app>.azurestaticapps.net`
- **API**: `https://<your-function-app>.azurewebsites.net`

## Testing the Application

### 1. Access the Frontend

Navigate to your Static Web App URL in a browser. You'll see the AI Travel Planner interface.

### 2. Create a Travel Plan

1. Enter your travel preferences in the chat interface:
   - Destination preferences
   - Travel dates
   - Budget range
   - Activities of interest

2. Submit your request and watch the durable agents workflow in action:
   - **Destination Agent**: Analyzes your preferences and suggests destinations
   - **Itinerary Agent**: Creates detailed day-by-day plans
   - **Local Guide Agent**: Adds insider tips and local attractions

3. Review the generated travel plan and approve it to complete the workflow

### 3. Monitor Workflow Execution

View the Durable Task Scheduler dashboard to monitor your workflow:

**Dashboard URL**: `https://dashboard.durabletask.io/`

> **Note**: You'll need to configure the dashboard with your Azure scheduler details. In the dashboard, enter your subscription ID, resource group, and scheduler name to connect to your deployed application.

The dashboard shows:
- Active workflow instances
- Execution history
- Task completion status
- Error details (if any)
- Performance metrics

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

```bash
docker run -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:latest
```

### 3. Configure Local Settings

Copy `api/local.settings.json.template` to `api/local.settings.json` and update the values:

```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "python",
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

### 4. Install Python Dependencies

```bash
cd api
pip install -r requirements.txt
```

### 5. Start the Backend

```bash
cd api
func start
```

### 6. Start the Frontend

```bash
cd frontend
npm install
npm start
```

The application will be available at `http://localhost:3000`.

## Clean Up

To remove all Azure resources and avoid ongoing charges:

```bash
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
