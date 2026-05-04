# ============================================================================
# Agent-Directed Workflows — Durable Task SDK (Python)
#
# FastAPI web app with a Durable Task worker running in-process.
# Each chat session is a durable entity — a persistent agent with durable state.
# ============================================================================

import asyncio
import json
import logging
import os
import threading
import uuid

from contextlib import asynccontextmanager

import redis as redis_lib
import uvicorn
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel

from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker
from durabletask.entities import EntityInstanceId

from chat_agent_entity import chat_agent_entity

# ─── Configuration ───

CONNECTION_STRING = os.environ.get(
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING",
    "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None",
)
REDIS_CONNECTION = os.environ.get("REDIS_CONNECTION_STRING", "localhost:6379")

# Parse connection string
def _parse_connection_string(conn_str: str) -> dict:
    parts = {}
    for part in conn_str.split(";"):
        if "=" in part:
            key, _, value = part.partition("=")
            # Handle values that contain '=' (like URLs with query params)
            parts[key.strip()] = value.strip() if key.strip() != "Endpoint" else part[len("Endpoint="):].strip()
    return parts

_config = _parse_connection_string(CONNECTION_STRING)
_endpoint = _config.get("Endpoint", "http://localhost:8080")
_taskhub = _config.get("TaskHub", "default")
_is_local = "localhost" in _endpoint or "127.0.0.1" in _endpoint
_credential = None if _is_local else __import__("azure.identity", fromlist=["DefaultAzureCredential"]).DefaultAzureCredential()

logging.basicConfig(
    level=logging.WARNING,
    format="%(asctime)s %(name)s %(levelname)s %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
)
logging.getLogger("AgentDirectedWorkflows").setLevel(logging.INFO)
logger = logging.getLogger("AgentDirectedWorkflows")

# ─── Durable Task Worker ───

worker = DurableTaskSchedulerWorker(
    host_address=_endpoint,
    secure_channel=not _is_local,
    taskhub=_taskhub,
    token_credential=_credential,
)
worker.add_entity(chat_agent_entity)

client = DurableTaskSchedulerClient(
    host_address=_endpoint,
    secure_channel=not _is_local,
    taskhub=_taskhub,
    token_credential=_credential,
)

# ─── FastAPI App ───

@asynccontextmanager
async def lifespan(app):
    """Start the Durable Task worker on startup, stop on shutdown."""
    worker.start()
    logger.info("Durable Task worker started")
    yield
    worker.stop()


app = FastAPI(title="Agent-Directed Workflows", lifespan=lifespan)


class ChatRequest(BaseModel):
    message: str


@app.post("/chat/{session_id}")
async def send_message(session_id: str, req: ChatRequest, request: Request):
    """Send a message to the agent. Streams SSE by default; add ?stream=false for JSON."""
    stream = request.query_params.get("stream", "true").lower() != "false"
    correlation_id = uuid.uuid4().hex
    channel = f"chat:{session_id}:{correlation_id}"

    r = redis_lib.Redis.from_url(f"redis://{REDIS_CONNECTION}")
    pubsub = r.pubsub()
    pubsub.subscribe(channel)

    # Signal the entity (fire-and-forget) — run in thread since it's blocking gRPC
    entity_id = EntityInstanceId("chat_agent_entity", session_id)
    await asyncio.to_thread(
        lambda: client.signal_entity(
            entity_id,
            "message",
            input={"message": req.message, "correlation_id": correlation_id},
        )
    )

    def _poll_next():
        """Block until we get a Redis message (runs in thread pool)."""
        return pubsub.get_message(ignore_subscribe_messages=True, timeout=120)

    if stream:
        async def event_stream():
            try:
                while True:
                    msg = await asyncio.to_thread(_poll_next)
                    if msg and msg["type"] == "message":
                        data = msg["data"].decode("utf-8") if isinstance(msg["data"], bytes) else msg["data"]
                        yield f"data: {data}\n\n"
                        try:
                            parsed = json.loads(data)
                            if parsed.get("type") in ("done", "error"):
                                break
                        except json.JSONDecodeError:
                            pass
            finally:
                pubsub.unsubscribe(channel)
                pubsub.close()
                r.close()

        return StreamingResponse(event_stream(), media_type="text/event-stream",
                                 headers={"Cache-Control": "no-cache"})
    else:
        # Non-streaming: collect all chunks and return JSON
        full_response = ""
        try:
            while True:
                msg = await asyncio.to_thread(_poll_next)
                if msg and msg["type"] == "message":
                    data = msg["data"].decode("utf-8") if isinstance(msg["data"], bytes) else msg["data"]
                    try:
                        parsed = json.loads(data)
                        if parsed.get("type") == "chunk":
                            full_response += parsed.get("content", "")
                        if parsed.get("type") == "error":
                            return JSONResponse(
                                {"sessionId": session_id, "error": parsed.get("content", "")},
                                status_code=500,
                            )
                        if parsed.get("type") in ("done", "error"):
                            break
                    except json.JSONDecodeError:
                        pass
        finally:
            pubsub.unsubscribe(channel)
            pubsub.close()
            r.close()

        return JSONResponse({"sessionId": session_id, "message": full_response})


@app.get("/chat/{session_id}/history")
async def get_history(session_id: str):
    """Get the full conversation history for a session."""
    entity_id = EntityInstanceId("chat_agent_entity", session_id)
    entity = await asyncio.to_thread(client.get_entity, entity_id)
    if entity is None:
        return JSONResponse({"error": "Session not found"}, status_code=404)
    raw_state = entity.get_state()
    if isinstance(raw_state, str):
        import json as _json
        state = _json.loads(raw_state)
    elif isinstance(raw_state, dict):
        state = raw_state
    else:
        state = {"messages": []}
    return JSONResponse({"sessionId": session_id, "history": state.get("messages", [])})


@app.post("/chat/{session_id}/reset")
async def reset_session(session_id: str):
    """Clear a session's conversation history."""
    entity_id = EntityInstanceId("chat_agent_entity", session_id)
    await asyncio.to_thread(client.signal_entity, entity_id, "reset")
    return JSONResponse({"sessionId": session_id, "status": "reset"})


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=5000, log_level="warning")
