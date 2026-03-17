"""Run a Copilot SDK agent with tools as a Durable Task activity.

The key insight: Copilot SDK handles the entire agentic loop internally.
No manual while-loop, no tool call parsing, no conversation history management.
"""
from __future__ import annotations

import asyncio

from copilot import CopilotClient, PermissionHandler

from tools import convert_units, lookup_word

SYSTEM_PROMPT = (
    "You are a helpful assistant with access to a dictionary and a unit converter. "
    "Use your tools when they help answer the question. Respond concisely."
)


def run_agent(ctx, user_input: str) -> str:
    del ctx

    async def _run_agent() -> str:
        client = CopilotClient()
        await client.start()
        try:
            session = await client.create_session({
                "model": "gpt-5.4",
                "on_permission_request": PermissionHandler.approve_all,
                "system_message": {"content": SYSTEM_PROMPT},
                "tools": [lookup_word, convert_units],
            })
            try:
                response = await session.send_and_wait({"prompt": user_input})
                return response.data.content if response else "No response"
            finally:
                await session.disconnect()
        finally:
            await client.stop()

    return asyncio.run(_run_agent())
