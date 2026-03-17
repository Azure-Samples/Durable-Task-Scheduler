from __future__ import annotations

import asyncio
from typing import Any

from copilot import CopilotClient, PermissionHandler
from durabletask import task


def send_agent_message(_ctx: task.ActivityContext, payload: dict[str, str]) -> str:
    """Send a message to a Copilot SDK session.

    The session ID is deterministic so the orchestration can resume the same
    conversation after worker restarts.
    """

    async def _impl() -> str:
        session_id = payload["session_id"]
        prompt = payload["prompt"]

        client = CopilotClient()
        await client.start()
        try:
            try:
                session = await client.resume_session(
                    session_id,
                    {
                        "model": "gpt-5.4",
                        "on_permission_request": PermissionHandler.approve_all,
                    },
                )
            except Exception:
                session = await client.create_session(
                    {
                        "session_id": session_id,
                        "model": "gpt-5.4",
                        "on_permission_request": PermissionHandler.approve_all,
                    }
                )

            try:
                response = await session.send_and_wait({"prompt": prompt})
                if response and getattr(response, "data", None):
                    return getattr(response.data, "content", None) or "No response"
                return "No response"
            finally:
                await session.disconnect()
        finally:
            await client.stop()

    return asyncio.run(_impl())
