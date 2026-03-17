"""Reusable Durable Task activity wrapping the GitHub Copilot SDK.

This activity creates a CopilotClient session, sends a prompt, and returns
the agent's response.  Durable Task handles retries and persistence around
the Copilot SDK's reasoning loop.
"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass, field
from typing import Any

from copilot import CopilotClient, PermissionHandler
from copilot.tools import define_tool


@dataclass
class CopilotRequest:
    """Input for the Copilot SDK activity."""

    prompt: str
    model: str = "gpt-5.4"
    system_message: str | None = None
    tools: list[Any] = field(default_factory=list)
    custom_agents: list[dict[str, Any]] = field(default_factory=list)
    agent: str | None = None
    streaming: bool = False
    # BYOK provider config (optional — omit to use GitHub Copilot subscription)
    provider: dict[str, Any] | None = None
    timeout_seconds: float | None = None


@dataclass
class CopilotResponse:
    """Output from the Copilot SDK activity."""

    content: str
    messages: list[dict[str, Any]] = field(default_factory=list)


async def run_copilot_agent(request: CopilotRequest) -> CopilotResponse:
    """Run a Copilot SDK agent session and return the result.

    This is designed to be called from a Durable Task activity.  The entire
    Copilot reasoning loop (tool calls, planning, sub-agents) executes within
    a single activity invocation.  Durable Task's retry policy protects
    against transient failures.
    """
    client = CopilotClient()
    await client.start()

    try:
        session_config: dict[str, Any] = {
            "model": request.model,
            "on_permission_request": PermissionHandler.approve_all,
            "streaming": request.streaming,
        }

        if request.system_message:
            session_config["system_message"] = {
                "content": request.system_message,
            }

        if request.tools:
            session_config["tools"] = request.tools

        if request.custom_agents:
            session_config["custom_agents"] = request.custom_agents

        if request.agent:
            session_config["agent"] = request.agent

        if request.provider:
            session_config["provider"] = request.provider

        session = await client.create_session(session_config)

        try:
            send_coro = session.send_and_wait({"prompt": request.prompt})
            if request.timeout_seconds is not None:
                response = await asyncio.wait_for(send_coro, timeout=request.timeout_seconds)
            else:
                response = await send_coro

            content = ""
            if response and hasattr(response, "data") and hasattr(response.data, "content"):
                content = response.data.content or ""

            messages = []
            try:
                messages_coro = session.get_messages()
                if request.timeout_seconds is not None:
                    raw_messages = await asyncio.wait_for(messages_coro, timeout=min(request.timeout_seconds, 15))
                else:
                    raw_messages = await messages_coro
                messages = [
                    {"role": getattr(m, "role", "unknown"), "content": getattr(m, "content", "")}
                    for m in (raw_messages or [])
                ]
            except Exception:
                pass

            return CopilotResponse(content=content, messages=messages)

        finally:
            await session.disconnect()
    finally:
        await client.stop()


# ── Durable Task activity wrapper ──────────────────────────────────────

async def copilot_agent_activity(ctx: Any, request: CopilotRequest) -> CopilotResponse:
    """Durable Task activity that runs a Copilot SDK agent session."""
    return await run_copilot_agent(request)
