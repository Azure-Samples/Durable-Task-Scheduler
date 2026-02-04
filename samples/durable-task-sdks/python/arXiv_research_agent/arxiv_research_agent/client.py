"""FastAPI client for the arXiv Research Agent.

This module provides a REST API to start research agents and check their status.
"""

import asyncio
import json
import logging
import os
from contextlib import asynccontextmanager
from typing import AsyncIterator, List, Optional

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from durabletask import client as durable_client
from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from .auth import get_credential

# Map DurableTask status enum to human-readable strings
ORCHESTRATION_STATUS_MAP = {
    durable_client.OrchestrationStatus.PENDING: "PENDING",
    durable_client.OrchestrationStatus.RUNNING: "RUNNING",
    durable_client.OrchestrationStatus.COMPLETED: "COMPLETED",
    durable_client.OrchestrationStatus.FAILED: "FAILED",
    durable_client.OrchestrationStatus.TERMINATED: "TERMINATED",
    durable_client.OrchestrationStatus.SUSPENDED: "SUSPENDED",
}

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


# Pydantic models for API
class AgentStartRequest(BaseModel):
    """Request to start a new research agent."""
    topic: str
    max_iterations: int = 3
    
    @property
    def validated_max_iterations(self) -> int:
        """Ensure max_iterations is within bounds."""
        return max(1, min(10, self.max_iterations))


class AgentStartResponse(BaseModel):
    """Response when starting a new agent."""
    ok: bool
    instance_id: str


class AgentStatus(BaseModel):
    """Status of a research agent."""
    agent_id: str
    topic: str
    status: str
    created_at: Optional[str] = None
    iterations: Optional[int] = None
    report: Optional[str] = None


class AgentResult(BaseModel):
    """Result of a completed research agent."""
    topic: str
    iterations: int
    report: str
    findings_count: int


class TerminateResponse(BaseModel):
    """Response when terminating an agent."""
    ok: bool
    message: str


class HealthResponse(BaseModel):
    """Health check response."""
    status: str


# Global client
_client: Optional[DurableTaskSchedulerClient] = None


def get_client() -> DurableTaskSchedulerClient:
    """Get or create the DurableTask client."""
    global _client
    
    if _client is None:
        taskhub_name = os.getenv("TASKHUB", "default")
        endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
        credential = get_credential()
        
        logger.info(f"Creating client with endpoint={endpoint}, taskhub={taskhub_name}")
        
        _client = DurableTaskSchedulerClient(
            host_address=endpoint,
            secure_channel=endpoint != "http://localhost:8080",
            taskhub=taskhub_name,
            token_credential=credential
        )
    
    return _client


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    """Application lifespan handler."""
    logger.info("Starting arXiv Research Agent API...")
    yield
    logger.info("Shutting down arXiv Research Agent API...")


# Create FastAPI app
app = FastAPI(
    title="arXiv Research Agent API",
    description="An autonomous research agent for searching arXiv papers built with DurableTask SDK",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.post("/agents", response_model=AgentStartResponse)
async def start_agent(request: AgentStartRequest) -> AgentStartResponse:
    """Start a new research agent for the given topic.
    
    This endpoint starts a durable orchestration in the background that will:
    1. Search arXiv for papers on the topic
    2. Iteratively research related topics
    3. Make decisions about when to continue
    4. Synthesize findings into a final report
    """
    if not request.topic or not request.topic.strip():
        raise HTTPException(status_code=400, detail="Topic cannot be empty")
    
    try:
        client = get_client()
        
        # Schedule the orchestration
        instance_id = await asyncio.to_thread(
            client.schedule_new_orchestration,
            "arxiv_research_orchestrator",
            input={
                "topic": request.topic.strip(),
                "max_iterations": request.validated_max_iterations
            }
        )
        
        logger.info(f"Started literature review for topic '{request.topic}' with instance ID: {instance_id}")
        
        return AgentStartResponse(ok=True, instance_id=instance_id)
    
    except Exception as e:
        logger.error(f"Failed to start agent: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/agents", response_model=List[AgentStatus])
async def list_agents() -> List[AgentStatus]:
    """List all research agents and their statuses.
    
    Note: This endpoint currently returns an empty list because the DurableTask
    Python SDK does not support querying all orchestration instances. To check
    the status of a specific agent, use GET /agents/{instance_id} with the
    instance_id returned when starting the agent.
    
    For viewing all orchestrations, use the Durable Task Scheduler dashboard
    at http://localhost:8082 (emulator) or the Azure portal.
    """
    # DurableTask Python SDK does not currently support listing orchestrations.
    # Individual agents can be queried via GET /agents/{instance_id}.
    return []


@app.get("/agents/{instance_id}", response_model=AgentStatus)
async def get_agent_status(instance_id: str) -> AgentStatus:
    """Get the status of a specific research agent."""
    try:
        client = get_client()
        
        # Get orchestration state
        state = await asyncio.to_thread(client.get_orchestration_state, instance_id)
        
        if state is None:
            raise HTTPException(status_code=404, detail="Agent not found")

        status = ORCHESTRATION_STATUS_MAP.get(state.runtime_status, "UNKNOWN")
        
        # Parse output if completed
        report = None
        topic = ""
        iterations = 0
        
        if state.runtime_status == durable_client.OrchestrationStatus.COMPLETED and state.serialized_output:
            try:
                output = json.loads(state.serialized_output)
                report = output.get("report")
                topic = output.get("topic", "")
                iterations = output.get("iterations", 0)
            except json.JSONDecodeError:
                pass
        
        # Parse input for topic
        if state.serialized_input and not topic:
            try:
                input_data = json.loads(state.serialized_input)
                topic = input_data.get("topic", "")
            except json.JSONDecodeError:
                pass
        
        return AgentStatus(
            agent_id=instance_id,
            topic=topic,
            status=status,
            created_at=state.created_at.isoformat() if state.created_at else None,
            iterations=iterations,
            report=report
        )
    
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to get agent status: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/agents/{instance_id}/wait", response_model=AgentResult)
async def wait_for_agent(instance_id: str, timeout: int = 300) -> AgentResult:
    """Wait for a research agent to complete and return the result.
    
    Args:
        instance_id: The orchestration instance ID
        timeout: Maximum seconds to wait (default 300)
    """
    try:
        client = get_client()
        
        # Wait for completion
        state = await asyncio.to_thread(
            client.wait_for_orchestration_completion,
            instance_id,
            timeout=timeout
        )
        
        if state is None:
            raise HTTPException(status_code=408, detail="Timeout waiting for agent completion")
        
        if state.runtime_status == durable_client.OrchestrationStatus.FAILED:
            raise HTTPException(status_code=500, detail="Agent failed")
        
        if state.runtime_status != durable_client.OrchestrationStatus.COMPLETED:
            raise HTTPException(status_code=500, detail=f"Unexpected status: {state.runtime_status}")
        
        # Parse output
        if not state.serialized_output:
            raise HTTPException(status_code=500, detail="No output from orchestration")
        output = json.loads(state.serialized_output)
        
        return AgentResult(
            topic=output.get("topic", ""),
            iterations=output.get("iterations", 0),
            report=output.get("report", ""),
            findings_count=output.get("findings_count", 0)
        )
    
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to wait for agent: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.delete("/agents/{instance_id}", response_model=TerminateResponse)
async def terminate_agent(instance_id: str) -> TerminateResponse:
    """Terminate a running research agent."""
    try:
        client = get_client()

        # Terminate the orchestration
        await asyncio.to_thread(
            client.terminate_orchestration,
            instance_id,
            output="Terminated by user"
        )

        return TerminateResponse(ok=True, message=f"Agent {instance_id} terminated")
    
    except Exception as e:
        logger.error(f"Failed to terminate agent: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/health", response_model=HealthResponse)
async def health_check() -> HealthResponse:
    """Health check endpoint."""
    return HealthResponse(status="healthy")


if __name__ == "__main__":
    import uvicorn
    
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8000"))
    
    uvicorn.run(app, host=host, port=port)
