# ============================================================================
# Durable Agent Chat — powered by Durable Entities + Redis Streaming
#
# Each chat session is a durable entity that holds the full conversation
# history. When you send a message, the entity calls the LLM, executes any
# tool calls, and publishes response chunks to Redis pub/sub in real-time.
# The HTTP layer subscribes to the Redis channel and forwards chunks as
# Server-Sent Events (SSE) to the client.
# ============================================================================

import json
import logging
import os

import redis

from durabletask.entities import EntityContext

import tools

logger = logging.getLogger("AgentDirectedWorkflows")


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
    """Simple echo fallback that simulates streaming word by word."""
    words = f"Echo: {user_message}".split(" ")
    for word in words:
        yield word + " "


def _run_agent_loop(state: dict, user_message: str, channel: str, r: redis.Redis):
    """The core agent loop: call LLM, handle tool calls, publish chunks to Redis."""
    messages = state.get("messages", [])
    messages.append({"role": "user", "content": user_message})

    client, deployment = _get_chat_client()

    # Build the messages list for the LLM
    llm_messages = [{"role": "system", "content": "You are a helpful assistant."}]
    llm_messages.extend(messages)

    if client is None:
        # Echo fallback — no Azure OpenAI configured
        full_text = ""
        for chunk in _echo_response(user_message):
            full_text += chunk
            r.publish(channel, json.dumps({"type": "chunk", "content": chunk}))

        messages.append({"role": "assistant", "content": full_text.strip()})
        r.publish(channel, json.dumps({"type": "done"}))
        return messages

    # Real LLM loop with tool calling
    while True:
        response = client.chat.completions.create(
            model=deployment,
            messages=llm_messages,
            tools=tools.TOOL_DEFINITIONS if tools.TOOL_DEFINITIONS else None,
            stream=True,
        )

        full_text = ""
        tool_calls_acc = {}  # id -> {name, arguments}

        for chunk in response:
            if not chunk.choices:
                continue
            delta = chunk.choices[0].delta

            # Accumulate tool calls
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

            # Stream text chunks to Redis
            if delta.content:
                full_text += delta.content
                r.publish(channel, json.dumps({"type": "chunk", "content": delta.content}))

        if tool_calls_acc:
            # LLM wants to call tools — execute and loop
            tool_names = [tc["name"] for tc in tool_calls_acc.values()]
            logger.info("Executing tools: %s", ", ".join(tool_names))

            # Add assistant message with tool calls
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

            # Execute each tool and add results
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

        # Final text reply — save to durable state
        messages.append({"role": "assistant", "content": full_text})
        r.publish(channel, json.dumps({"type": "done"}))
        return messages


def chat_agent_entity(ctx: EntityContext, input):
    """Function-based durable entity for the chat agent.

    Operations:
      - message: Run the agent loop with a user message
      - get_history: Return the conversation history
      - reset: Clear the conversation
    """
    state = ctx.get_state(dict, {"messages": []})

    if ctx.operation == "message":
        request = input if isinstance(input, dict) else {}
        user_message = request.get("message", "Hello")
        correlation_id = request.get("correlation_id", "unknown")
        session_id = ctx.entity_id.key

        channel = f"chat:{session_id}:{correlation_id}"
        r = _get_redis()

        try:
            messages = _run_agent_loop(state, user_message, channel, r)
            state["messages"] = messages
            ctx.set_state(state)
        except Exception as ex:
            logger.error("Agent loop failed: %s", ex)
            r.publish(channel, json.dumps({"type": "error", "content": str(ex)}))

    elif ctx.operation == "get_history":
        return state.get("messages", [])

    elif ctx.operation == "reset":
        ctx.set_state({"messages": []})
