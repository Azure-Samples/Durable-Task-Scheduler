# ============================================================================
# Agent-Directed Workflows — Durable Functions (Python)
#
# Each chat session is a durable entity that holds the full conversation
# history. When you send a message, the entity calls the LLM, executes any
# tool calls, and publishes response chunks to Redis pub/sub in real-time.
# The HTTP layer subscribes to the Redis channel and forwards chunks.
# ============================================================================

import asyncio
import json
import logging
import os
import uuid

import azure.functions as func
import azure.durable_functions as df
import redis

import tools

app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

logger = logging.getLogger("AgentDirectedWorkflows")


# ─── Helpers ───

def _get_chat_client():
    """Create an OpenAI client, or None if not configured."""
    endpoint = os.environ.get("AZURE_OPENAI_ENDPOINT", "")
    deployment = os.environ.get("AZURE_OPENAI_DEPLOYMENT", "")
    if endpoint and deployment:
        from openai import AzureOpenAI
        from azure.identity import DefaultAzureCredential, get_bearer_token_provider
        token_provider = get_bearer_token_provider(
            DefaultAzureCredential(), "https://cognitiveservices.azure.com/.default"
        )
        return AzureOpenAI(
            azure_endpoint=endpoint,
            azure_ad_token_provider=token_provider,
            api_version="2024-10-21",
        ), deployment
    return None, None


def _get_redis():
    """Create a Redis connection."""
    conn_str = os.environ.get("REDIS_CONNECTION_STRING", "localhost:6379")
    return redis.Redis.from_url(f"redis://{conn_str}")


def _echo_response(user_message: str):
    """Simple echo fallback — yields words one at a time."""
    words = f"Echo: {user_message}".split(" ")
    for word in words:
        yield word + " "


def _run_agent_loop(state: dict, user_message: str, channel: str, r):
    """The core agent loop: call LLM, handle tool calls, publish chunks to Redis."""
    messages = state.get("messages", [])
    messages.append({"role": "user", "content": user_message})

    client, deployment = _get_chat_client()

    llm_messages = [{"role": "system", "content": "You are a helpful assistant."}]
    llm_messages.extend(messages)

    if client is None:
        full_text = ""
        for chunk in _echo_response(user_message):
            full_text += chunk
            r.publish(channel, json.dumps({"type": "chunk", "content": chunk}))
        messages.append({"role": "assistant", "content": full_text.strip()})
        r.publish(channel, json.dumps({"type": "done"}))
        return messages

    while True:
        response = client.chat.completions.create(
            model=deployment,
            messages=llm_messages,
            tools=tools.TOOL_DEFINITIONS if tools.TOOL_DEFINITIONS else None,
            stream=True,
        )

        full_text = ""
        tool_calls_acc = {}

        for chunk in response:
            if not chunk.choices:
                continue
            delta = chunk.choices[0].delta

            if delta.tool_calls:
                for tc in delta.tool_calls:
                    idx = tc.index
                    if idx not in tool_calls_acc:
                        tool_calls_acc[idx] = {"id": "", "name": "", "arguments": ""}
                    if tc.id:
                        tool_calls_acc[idx]["id"] = tc.id
                    if tc.function and tc.function.name:
                        tool_calls_acc[idx]["name"] = tc.function.name
                    if tc.function and tc.function.arguments:
                        tool_calls_acc[idx]["arguments"] += tc.function.arguments

            if delta.content:
                full_text += delta.content
                r.publish(channel, json.dumps({"type": "chunk", "content": delta.content}))

        if tool_calls_acc:
            tool_names = [tc["name"] for tc in tool_calls_acc.values()]
            logger.info("Executing tools: %s", ", ".join(tool_names))

            llm_messages.append({
                "role": "assistant",
                "tool_calls": [
                    {
                        "id": tc["id"],
                        "type": "function",
                        "function": {"name": tc["name"], "arguments": tc["arguments"]},
                    }
                    for tc in tool_calls_acc.values()
                ],
            })

            for tc in tool_calls_acc.values():
                try:
                    args = json.loads(tc["arguments"]) if tc["arguments"] else {}
                except json.JSONDecodeError:
                    args = {}
                result = tools.execute(tc["name"], args)
                llm_messages.append({
                    "role": "tool",
                    "tool_call_id": tc["id"],
                    "content": result,
                })
            continue

        messages.append({"role": "assistant", "content": full_text})
        r.publish(channel, json.dumps({"type": "done"}))
        return messages


# ─── Entity ───

@app.entity_trigger(context_name="context")
def chat_agent_entity(context):
    """Durable entity for chat agent sessions.

    Operations:
      - message: Run the agent loop with a user message
      - get_history: Return the conversation history
      - reset: Clear the conversation
    """
    state = context.get_state(lambda: {"messages": []})
    operation = context.operation_name

    if operation == "message":
        request = context.get_input()
        user_message = request.get("message", "Hello")
        correlation_id = request.get("correlation_id", "unknown")
        session_id = context.entity_key

        channel = f"chat:{session_id}:{correlation_id}"
        r = _get_redis()

        try:
            messages = _run_agent_loop(state, user_message, channel, r)
            state["messages"] = messages
            context.set_state(state)
        except Exception as ex:
            logger.error("Agent loop failed: %s", ex)
            r.publish(channel, json.dumps({"type": "error", "content": str(ex)}))

    elif operation == "get_history":
        context.set_result(state.get("messages", []))

    elif operation == "reset":
        context.set_state({"messages": []})


# ─── HTTP Endpoints ───

@app.route(route="chat/{sessionId}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def send_message(req: func.HttpRequest, client) -> func.HttpResponse:
    """Send a message to the agent. Returns JSON with the full response."""
    session_id = req.route_params.get("sessionId", "default")
    durable_client = client

    try:
        body = req.get_json()
        user_message = body.get("message", "Hello")
    except ValueError:
        user_message = "Hello"

    correlation_id = uuid.uuid4().hex
    channel = f"chat:{session_id}:{correlation_id}"

    # Subscribe to Redis BEFORE signaling the entity
    r = _get_redis()
    pubsub = r.pubsub()
    pubsub.subscribe(channel)

    # Signal the entity
    entity_id = df.EntityId("chat_agent_entity", session_id)
    await durable_client.signal_entity(
        entity_id,
        "message",
        {"message": user_message, "correlation_id": correlation_id},
    )

    # Collect response chunks (non-streaming for Functions HTTP model)
    # NOTE: Redis pubsub uses blocking I/O, so we run it in a thread to
    # avoid blocking the event loop (which would prevent entity dispatch).
    def _collect_response():
        result = {"response": "", "error": None}
        try:
            while True:
                msg = pubsub.get_message(ignore_subscribe_messages=True, timeout=120)
                if msg and msg["type"] == "message":
                    data = msg["data"].decode("utf-8") if isinstance(msg["data"], bytes) else msg["data"]
                    try:
                        parsed = json.loads(data)
                        if parsed.get("type") == "chunk":
                            result["response"] += parsed.get("content", "")
                        if parsed.get("type") == "error":
                            result["error"] = parsed.get("content", "")
                            break
                        if parsed.get("type") == "done":
                            break
                    except json.JSONDecodeError:
                        pass
        finally:
            pubsub.unsubscribe(channel)
            pubsub.close()
            r.close()
        return result

    collected = await asyncio.to_thread(_collect_response)

    if collected["error"]:
        return func.HttpResponse(
            json.dumps({"sessionId": session_id, "error": collected["error"]}),
            mimetype="application/json",
            status_code=500,
        )

    return func.HttpResponse(
        json.dumps({"sessionId": session_id, "message": collected["response"]}),
        mimetype="application/json",
    )


@app.route(route="chat/{sessionId}/history", methods=["GET"])
@app.durable_client_input(client_name="client")
async def get_history(req: func.HttpRequest, client) -> func.HttpResponse:
    """Get the full conversation history for a session."""
    session_id = req.route_params.get("sessionId", "default")
    durable_client = client

    entity_id = df.EntityId("chat_agent_entity", session_id)
    state = await durable_client.read_entity_state(entity_id)

    if not state.entity_exists:
        return func.HttpResponse(
            json.dumps({"error": "Session not found"}),
            mimetype="application/json",
            status_code=404,
        )

    entity_state = state.entity_state or {"messages": []}
    if isinstance(entity_state, str):
        entity_state = json.loads(entity_state)
    return func.HttpResponse(
        json.dumps({"sessionId": session_id, "history": entity_state.get("messages", [])}),
        mimetype="application/json",
    )


@app.route(route="chat/{sessionId}/reset", methods=["POST"])
@app.durable_client_input(client_name="client")
async def reset_session(req: func.HttpRequest, client) -> func.HttpResponse:
    """Clear a session's conversation history."""
    session_id = req.route_params.get("sessionId", "default")
    durable_client = client

    entity_id = df.EntityId("chat_agent_entity", session_id)
    await durable_client.signal_entity(entity_id, "reset")

    return func.HttpResponse(
        json.dumps({"sessionId": session_id, "status": "reset"}),
        mimetype="application/json",
    )
