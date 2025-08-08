# AI Agent Travel Planner - Python Implementation

This is a Python implementation of the AI Agent Travel Planner using Azure Durable Functions and Azure OpenAI, with a React frontend for a complete user experience.

## Features

- **Multi-Agent Travel Planning**: Uses three specialized Azure OpenAI agents:
  - **Destination Recommender Agent**: Suggests destinations based on preferences
  - **Itinerary Planner Agent**: Creates detailed daily itineraries  
  - **Local Recommendations Agent**: Provides authentic local insights
- **Durable Functions Orchestration**: Coordinates multiple AI agents in a reliable workflow
- **Human Approval Workflow**: Implements approval/rejection workflow for travel plans
- **Booking Integration**: Simulated booking process after plan approval
- **React Frontend**: Interactive chat interface with real-time progress updates
- **CORS Support**: Configured for frontend integration

## Architecture

The application uses a multi-agent orchestration pattern:

```
HTTP Request → Orchestrator → Destinations Agent → Itinerary Agent → Local Tips Agent → Approval → Booking → Response
```

## Prerequisites

- Python 3.8+
- Node.js 16+ (for frontend)
- Azure Functions Core Tools
- Azure OpenAI resource with GPT-4 deployment
- Azure subscription

## Setup

1. **Clone and navigate to the project**:
   ```bash
   cd samples/durable-functions/python/ai-agent-travel-planner
   ```

2. **Configure environment variables**:
   - Copy `api/local.settings.json.template` to `api/local.settings.json`
   - Update the values with your Azure OpenAI credentials:
   ```json
   {
     "IsEncrypted": false,
     "Values": {
       "AzureWebJobsStorage": "UseDevelopmentStorage=true",
       "FUNCTIONS_WORKER_RUNTIME": "python",
       "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None;",
       "TASKHUB_NAME": "default",
       "AZURE_OPENAI_ENDPOINT": "https://your-openai-service.openai.azure.com/",
       "AZURE_OPENAI_DEPLOYMENT": "your-gpt-deployment-name",
       "AZURE_OPENAI_API_KEY": "your-openai-api-key-here",
       "ALLOWED_ORIGINS": "http://localhost:3000"
     }
   }
   ```

3. **Install Python dependencies**:
   ```bash
   cd api
   pip install -r requirements.txt
   ```

4. **Start the backend**:
   ```bash
   cd api
   func start
   ```

5. **Start the frontend** (in a new terminal):
   ```bash
   cd frontend
   npm install
   npm start
   ```

6. **Open your browser** to `http://localhost:3000`

## API Endpoints

### Core Functions
- `POST /api/travel-planner` - Start travel planning process
- `GET /api/travel-planner/status/{instance_id}` - Check planning status  
- `POST /api/travel-planner/approve/{instance_id}` - Approve or reject travel plan

### Durable Functions Management (for testing)
- `POST /api/orchestrators/TravelPlannerOrchestrator` - Direct orchestration start
- `POST /runtime/webhooks/durabletask/instances/{instanceId}/raiseEvent/approval_response` - Send approval events

## Usage

1. **Web Interface**: Use the React frontend for the full interactive experience at `http://localhost:3000`
2. **API Testing**: Use the provided `test-api.http` file with VS Code REST Client extension
3. **Direct Testing**: Make HTTP requests to the API endpoints

### Simple API Workflow

The `test-api.http` file contains a simplified 3-step workflow:

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

The AI agents use optimized prompts for concise responses to stay within Azure Functions payload limits:

- **Destination Agent**: Recommends 3 destinations with brief descriptions and match scores
- **Itinerary Agent**: Creates daily plans with max 3 activities per day and cost estimates
- **Local Agent**: Provides max 3 attractions and restaurants with insider tips

## Development

### File Structure
```
├── api/                          # Azure Functions backend
│   ├── function_app.py           # Main orchestration and HTTP triggers
│   ├── ai_services/
│   │   ├── agent_service.py      # AI agent implementations
│   ├── models/
│   │   ├── travel_models.py      # Data models
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

## Troubleshooting

- **Payload Size Errors**: The prompts are optimized to stay under 16KB limit
- **Authentication**: Ensure Azure OpenAI API key is correctly configured
- **CORS Issues**: Frontend origin is configured in local.settings.json
- **Function Timeout**: Long-running orchestrations handle timeouts gracefully

## License

This sample is provided under the MIT License.
