# arXiv Research Agent

An autonomous academic research agent that searches arXiv papers, built with the **DurableTask Python SDK** and **Azure Durable Task Scheduler**.

This example demonstrates how to build reliable, durable AI agents using durable orchestrations. The agent performs automated research: starting with a research topic, it iteratively searches arXiv for relevant papers, analyzes abstracts, identifies research gaps, and synthesizes findings into a comprehensive academic report with proper citations.

Because the agent is implemented as a durable orchestration, it can recover from any failure and continue the research from where it left off, ensuring no work is lost.

## How It Works

The agent performs automated academic research for any topic you provide:

1. **Search**: Queries arXiv's academic paper database for your research topic
2. **Analyze**: Extracts insights from paper metadata, abstracts, and categories using LLM analysis
3. **Identify Gaps**: Analyzes current findings to identify unexplored research directions
4. **Iterate**: Generates follow-up queries to explore related areas and fill gaps
5. **Synthesize**: Compiles all findings into an academic research report with inline citations

## Features

- **Durable Orchestrations**: The research workflow is fully durable - if it fails at any point, it automatically resumes from where it left off
- **Continue-as-New Pattern**: Uses the `continue_as_new` pattern to prevent unbounded history growth, making it suitable for long-running research workflows
- **Academic Focus**: Designed specifically for academic paper research with proper arXiv citations and research gap analysis
- **arXiv Integration**: Direct integration with arXiv API supporting search by keyword, category, and paper ID
- **LLM-Powered Analysis**: Uses Azure OpenAI to analyze papers, identify research gaps, and write research reports
- **REST API**: FastAPI-based API for starting research and checking status
- **Sub-orchestrations**: Uses the sub-orchestration pattern for modular paper research
- **Rate Limiting**: Built-in rate limiting and retry logic for arXiv API compliance

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     FastAPI Client (client.py)                      │
│  POST /agents - Start research      GET /agents/{id} - Get status   │
└────────────────────────────────────┬────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  Durable Task Scheduler (Emulator/Azure)            │
│                     Orchestration State Management                  │
└────────────────────────────────────┬────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Worker (worker.py)                           │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │           arxiv_research_orchestrator (Main)                 │   │
│  │  • Uses continue_as_new for iterative research               │   │
│  │  • Identifies research gaps and generates queries            │   │
│  │  • Synthesizes final research report                         │   │
│  │                                                              │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │     paper_research_orchestrator (Sub-orchestration)  │    │   │
│  │  │  • Search arXiv for papers                           │    │   │
│  │  │  • Analyze papers and extract insights               │    │   │
│  │  └──────────────────────────────────────────────────────┘    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Activities:                                                        │
│  • search_arxiv_activity - Search arXiv API for papers              │
│  • analyze_papers_activity - LLM analyzes papers                    │
│  • identify_research_gaps_activity - LLM identifies gaps            │
│  • decide_continuation_activity - LLM decides to continue/stop      │
│  • synthesize_research_activity - LLM writes final report           │
└─────────────────────────────────────────────────────────────────────┘
```

## Research Orchestration

The core of the agent is the main research orchestration using the **continue_as_new** pattern. This pattern prevents unbounded orchestration history growth by restarting the orchestration with updated state after each iteration.

```python
def arxiv_research_orchestrator(ctx: task.OrchestrationContext, input: Dict[str, Any]):
    """
    This agent performs automated research using continue_as_new:
    1. Executes one research iteration per orchestration instance
    2. Calls continue_as_new with updated state to proceed to next iteration
    3. Returns final result when max iterations reached or early termination
    """
    # Extract state (supports both initial call and continue_as_new)
    topic = input["topic"]
    max_iterations = input.get("max_iterations", 3)
    current_iteration = input.get("current_iteration", 0)
    all_findings = input.get("all_findings", [])
    current_query = input.get("current_query", topic)

    # Check if we've reached max iterations
    if current_iteration >= max_iterations:
        final_report = yield ctx.call_activity("synthesize_research_activity", ...)
        return {"topic": topic, "iterations": current_iteration, "report": final_report}

    current_iteration += 1

    # Research papers using a sub-orchestration
    analysis = yield ctx.call_sub_orchestrator(
        "paper_research_orchestrator",
        input={"main_topic": topic, "query": current_query}
    )
    all_findings.append(analysis)

    # Decide whether to continue the research
    should_continue = yield ctx.call_activity("decide_continuation_activity", ...)
    if not should_continue:
        final_report = yield ctx.call_activity("synthesize_research_activity", ...)
        return {"topic": topic, "iterations": current_iteration, "report": final_report}

    # Identify research gaps for next query
    follow_up_query = yield ctx.call_activity("identify_research_gaps_activity", ...)
    if not follow_up_query:
        final_report = yield ctx.call_activity("synthesize_research_activity", ...)
        return {"topic": topic, "iterations": current_iteration, "report": final_report}

    # Continue as new - resets history, preserves state
    ctx.continue_as_new({
        "topic": topic,
        "max_iterations": max_iterations,
        "current_iteration": current_iteration,
        "all_findings": all_findings,
        "current_query": follow_up_query
    })
```

### Why continue_as_new?

The `continue_as_new` pattern is ideal for long-running iterative workflows because:
- **Prevents history bloat**: Each iteration starts with fresh history
- **Maintains durability**: State is preserved across restarts
- **Avoids platform limits**: Long orchestrations won't hit history size limits

## Paper Research Sub-Orchestration

Each iteration calls a sub-orchestration that searches arXiv for papers and analyzes them.

```python
def paper_research_orchestrator(ctx: task.OrchestrationContext, input: Dict[str, Any]):
    """Research papers for a specific query within the research workflow."""
    main_topic = input["main_topic"]
    query = input["query"]
    
    # Step 1: Search arXiv for papers
    papers = yield ctx.call_activity("search_arxiv_activity", input=query)
    
    if not papers:
        return {"query": query, "insights": [], "relevance_score": 0, ...}
    
    # Step 2: Analyze papers and extract academic insights
    analysis = yield ctx.call_activity(
        "analyze_papers_activity",
        input={"topic": main_topic, "query": query, "papers": papers}
    )
    
    return analysis
```

## arXiv API Integration

The agent uses arXiv's official API with built-in rate limiting (3s between requests) and retry logic for 429/503 errors.

```python
def search_arxiv(query: str, max_results: int = 30) -> List[Dict[str, Any]]:
    """Search arXiv for papers matching the query."""
    # Supports arXiv query syntax
    # Returns: arxiv_id, title, authors, abstract, categories, pdf_url, etc.

def search_arxiv_by_category(category: str, query: str = "") -> List[Dict[str, Any]]:
    """Search within a specific arXiv category (e.g., cs.AI, cs.LG)."""

def get_paper_by_id(arxiv_id: str) -> Optional[Dict[str, Any]]:
    """Get a specific paper by its arXiv ID."""
```

## API Endpoints

```python
@app.post("/agents")
async def start_agent(request: AgentStartRequest):
    """Start a research job in the background"""
    instance_id = await client.schedule_new_orchestration(
        "arxiv_research_orchestrator",
        input={"topic": request.topic, "max_iterations": request.max_iterations}
    )
    return {"ok": True, "instance_id": instance_id}

@app.get("/agents/{instance_id}")
async def get_agent_status(instance_id: str):
    """Get the status of a research job"""
    state = await client.get_orchestration_state(instance_id)
    return {"status": state.runtime_status.name, ...}

@app.get("/agents/{instance_id}/wait")
async def wait_for_agent(instance_id: str, timeout: int = 300):
    """Wait for completion and return the research report"""
```

---

## Try It Yourself

### Prerequisites

1. **Python 3.9+**
2. **Docker** (for running the Durable Task Scheduler emulator)
3. **Azure OpenAI** endpoint and credentials

### 1. Clone and Install Dependencies

```bash
cd arxiv-research-agent

# Create virtual environment
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

### 2. Configure Environment Variables

```bash
cp .env.example .env
# Edit .env with your Azure OpenAI credentials
```

Required variables:
- `AZURE_OPENAI_ENDPOINT` - Your Azure OpenAI endpoint
- `AZURE_OPENAI_API_KEY` - API key (or leave unset for Entra ID auth)
- `AZURE_OPENAI_DEPLOYMENT` - Model deployment name (default: gpt-5.2)

### 3. Start the Durable Task Scheduler Emulator

```bash
# Pull the emulator image
docker pull mcr.microsoft.com/dts/dts-emulator:latest

# Run the emulator
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
```

The emulator dashboard is available at http://localhost:8082

## Running the Agent

### Start the Worker

In one terminal:

```bash
source .venv/bin/activate
python -m arxiv_research_agent.worker
```

### Start the API Server

In another terminal:

```bash
source .venv/bin/activate
python -m arxiv_research_agent.client
```

The API is now available at http://localhost:8000

### API Documentation

Open http://localhost:8000/docs for the interactive Swagger UI.

## Usage

### Start a Research Job

```bash
curl -X POST http://localhost:8000/agents \
  -H "Content-Type: application/json" \
  -d '{"topic": "transformer attention mechanisms in NLP", "max_iterations": 3}'
```

Response:
```json
{
  "ok": true,
  "instance_id": "abc123-def456-..."
}
```

### Check Status

```bash
curl http://localhost:8000/agents/{instance_id}
```

### Wait for Completion

```bash
curl "http://localhost:8000/agents/{instance_id}/wait?timeout=300"
```

This returns the final research report once complete.

### View in Dashboard

Open http://localhost:8082 to view orchestration progress in the Durable Task Scheduler dashboard.

<img alt="Image" src="assets/image.png" />

## Using Azure Durable Task Scheduler (Production)

For production deployment, use Azure Durable Task Scheduler instead of the emulator:

### 1. Install Azure CLI Extension

```bash
az upgrade
az extension add --name durabletask --allow-preview true
```

### 2. Create Scheduler Resources

```bash
# Create resource group
az group create --name my-resource-group --location eastus

# Create scheduler
az durabletask scheduler create \
    --resource-group my-resource-group \
    --name my-scheduler \
    --ip-allowlist '["0.0.0.0/0"]' \
    --sku-name "Dedicated" \
    --sku-capacity 1

# Create task hub
az durabletask taskhub create \
    --resource-group my-resource-group \
    --scheduler-name my-scheduler \
    --name "my-taskhub"

# Grant permissions
subscriptionId=$(az account show --query "id" -o tsv)
loggedInUser=$(az account show --query "user.name" -o tsv)

az role assignment create \
    --assignee $loggedInUser \
    --role "Durable Task Data Contributor" \
    --scope "/subscriptions/$subscriptionId/resourceGroups/my-resource-group/providers/Microsoft.DurableTask/schedulers/my-scheduler/taskHubs/my-taskhub"
```

### 3. Set Environment Variables

```bash
export ENDPOINT=$(az durabletask scheduler show \
    --resource-group my-resource-group \
    --name my-scheduler \
    --query "properties.endpoint" \
    --output tsv)

export TASKHUB="my-taskhub"
```

## Project Structure

```
arxiv-research-agent/
├── arxiv_research_agent/
│   ├── __init__.py
│   ├── activities.py      # DurableTask activities
│   ├── orchestrations.py  # DurableTask orchestrations
│   ├── worker.py          # Worker process
│   ├── client.py          # FastAPI REST API
│   ├── models.py          # Data models
│   ├── llm.py             # Azure OpenAI LLM utilities
│   └── arxiv_api.py       # arXiv API client with rate limiting
├── tests/                 # Test suite
├── requirements.txt
├── pyproject.toml
├── .env.example
└── README.md
```

## arXiv Categories

The agent can search across all arXiv categories. Some popular ones include:

- **cs.AI** - Artificial Intelligence
- **cs.LG** - Machine Learning
- **cs.CL** - Computation and Language (NLP)
- **cs.CV** - Computer Vision
- **cs.NE** - Neural and Evolutionary Computing
- **stat.ML** - Machine Learning (Statistics)
- **physics.hep-th** - High Energy Physics - Theory
- **math.CO** - Combinatorics
- **quant-ph** - Quantum Physics

## Disclaimer

[arXiv](https://arxiv.org) is an independent, open-access archive operated by Cornell University and is not affiliated with Microsoft. This sample application is provided for demonstration purposes only.

## License

MIT
