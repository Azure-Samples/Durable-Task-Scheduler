# AI Travel Planner with Durable Agents - Python

A travel planning application that demonstrates how to build **durable AI agents** using the [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/user-guide/agents/agent-types/durable-agent/create-durable-agent). The application coordinates multiple specialized AI agents to create comprehensive, personalized travel plans through a structured workflow.

## Overview

This sample showcases an agentic workflow where specialized AI agents collaborate to plan travel experiences. Each agent focuses on a specific aspect of travel planning—destination recommendations, itinerary creation, and local insights—orchestrated by the Durable Task extension for reliability and state management.

### Why Durable Agents?

Traditional AI agents can be unpredictable and inconsistent. The [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/user-guide/agents/agent-types/durable-agent/create-durable-agent) solves this by providing:

- **Deterministic workflows**: Pre-defined steps ensure consistent, high-quality results
- **Built-in resilience**: Automatic state persistence and recovery from failures
- **Human-in-the-loop**: Native support for approval workflows before booking
- **Scalability**: Serverless execution that scales with demand

## Architecture

The application uses a multi-agent orchestration pattern:

```
HTTP Request → Orchestrator → Destination Agent → Itinerary Agent → Local Tips Agent → Approval → Booking → Response
```

### Workflow

1. **User Request** → User submits travel preferences via React frontend
2. **Destination Recommendation** → AI agent analyzes preferences and suggests destinations
3. **Itinerary Planning** → AI agent creates detailed day-by-day plans (with currency conversion tools)
4. **Local Recommendations** → AI agent adds insider tips and attractions
5. **Approval** → User reviews and approves the plan (human-in-the-loop)
6. **Booking** → Upon approval, booking process completes

### Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Python 3.10+, Azure Functions |
| AI Framework | Microsoft Agent Framework with Durable Task Extension |
| Orchestration | Durable Task Scheduler |
| AI Model | Azure OpenAI (GPT-4o-mini) |
| Frontend | React |
| Hosting | Azure Static Web Apps, Azure Functions |
| Infrastructure | Bicep, Azure Developer CLI (azd) |

## Features

- **Multi-Agent Travel Planning**: Uses three specialized durable agents:
  - **DestinationRecommenderAgent**: Suggests destinations based on preferences
  - **ItineraryPlannerAgent**: Creates detailed daily itineraries with currency conversion
  - **LocalRecommendationsAgent**: Provides authentic local insights
- **Automatic Session Management**: Agent state is automatically persisted and survives failures
- **Currency Conversion Tools**: Itinerary agent uses real exchange rates for cost estimates
- **Human Approval Workflow**: Implements approval/rejection workflow for travel plans
- **Structured Responses**: Uses Pydantic models for type-safe agent responses
- **React Frontend**: Interactive chat interface with real-time progress updates

## Prerequisites

- Python 3.10+
- Node.js 18+ (for frontend)
- [Azure Functions Core Tools v4](https://docs.microsoft.com/en-us/azure/azure-functions/functions-run-local)
- [Docker](https://www.docker.com/get-started) (for local Durable Task Scheduler emulator)
- Azure OpenAI resource with GPT-4o-mini deployment
- Azure subscription

## Local Development

### 1. Start Azure Storage Emulator

```bash
npm install -g azurite
azurite --silent --location ./azurite &
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
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None;",
    "TASKHUB_NAME": "default",
    "AZURE_OPENAI_ENDPOINT": "https://your-openai-service.openai.azure.com/",
    "AZURE_OPENAI_DEPLOYMENT_NAME": "gpt-4o-mini",
    "ALLOWED_ORIGINS": "http://localhost:3000"
  }
}
```

> **Note**: The application uses `DefaultAzureCredential` for authentication. Run `az login` before starting the application.

### 4. Install Python dependencies

```bash
cd api
pip install -r requirements.txt
```

### 5. Start the backend

```bash
cd api
func start
```

### 6. Start the frontend (in a new terminal)

```bash
cd frontend
npm install
npm start
```

The application will be available at `http://localhost:3000`.

## API Endpoints

### Orchestration Endpoints
- `POST /api/travel-planner` - Start travel planning orchestration
- `GET /api/travel-planner/status/{instance_id}` - Check planning status  
- `POST /api/travel-planner/approve/{instance_id}` - Approve or reject travel plan

### Agent Endpoints (auto-generated by AgentFunctionApp)
- `POST /api/agents/{agentName}/run` - Run a single agent interaction
- `POST /api/agents/{agentName}/threads` - Create a new thread and run
- `POST /api/agents/{agentName}/threads/{threadId}` - Continue an existing thread

## Usage

1. **Web Interface**: Use the React frontend for the full interactive experience at `http://localhost:3000`
2. **API Testing**: Use the provided `test-api.http` file with VS Code REST Client extension
3. **Direct Testing**: Make HTTP requests to the API endpoints

### Simple API Workflow

1. **Create Travel Plan**:
   ```http
   POST http://localhost:7071/api/travel-planner
   ```
   Response includes an instance ID for tracking.

2. **Check Status**:
   ```http
   GET http://localhost:7071/api/travel-planner/status/{instance_id}
   ```
   Monitor progress until status shows "WaitingForApproval".

3. **Approve Plan**:
   ```http
   POST http://localhost:7071/api/travel-planner/approve/{instance_id}
   ```
   Triggers booking and completes the workflow.

## Agent Configuration

The AI agents use the Durable Task extension pattern for reliable, stateful execution:

- **DestinationRecommenderAgent**: Recommends 3 destinations with descriptions and match scores
- **ItineraryPlannerAgent**: Creates daily plans with currency conversion tools for cost estimates
- **LocalRecommendationsAgent**: Provides attractions, restaurants, and insider tips

## Development

### File Structure
```
├── api/                          # Azure Functions backend
│   ├── function_app.py           # Durable agents and orchestration
│   ├── tools/
│   │   ├── currency_converter.py # Currency conversion tool for agents
│   ├── models/
│   │   ├── travel_models.py      # Pydantic data models
│   ├── requirements.txt          # Python dependencies
│   ├── host.json                 # Function host configuration
│   ├── local.settings.json.template # Configuration template
│   └── azure.yaml                # Azure deployment config
├── frontend/                     # React application
│   ├── src/
│   │   ├── components/
│   │   │   ├── ChatInterface.js  # Main chat interface
│   │   │   └── ProgressTracker.js # Progress tracking component
│   │   └── App.js                # Main React app
│   ├── package.json              # Node.js dependencies
│   └── public/                   # Static assets
├── test-api.http                 # HTTP test requests
└── README.md                     # Project documentation
```

### Testing
- Use `test-api.http` for simple API testing with a 3-step workflow
- Frontend includes real-time status updates and approval workflow
- Orchestration supports both approval and rejection flows
- Monitor workflow in Durable Task Scheduler dashboard: `https://dashboard.durabletask.io/`

## Learn More

- [Durable Task Extension for Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/user-guide/agents/agent-types/durable-agent/create-durable-agent)
- [Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/overview/agent-framework-overview)
- [Azure Durable Task Scheduler](https://learn.microsoft.com/en-us/azure/azure-functions/durable/durable-task-scheduler/quickstart-durable-task-scheduler)
- [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/)

## Troubleshooting

- **Authentication**: Uses `DefaultAzureCredential` - run `az login` before starting
- **CORS Issues**: Frontend origin is configured in local.settings.json
- **Durable Task Scheduler**: Ensure emulator is running on port 8080

## License

This sample is provided under the MIT License.
