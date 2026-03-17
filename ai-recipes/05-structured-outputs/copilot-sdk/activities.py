"""Use the Copilot SDK to parse messy review notes into validated JSON."""
from __future__ import annotations

import asyncio
import json

from copilot import CopilotClient, PermissionHandler

from tools import ReviewList, submit_reviews

SYSTEM_PROMPT = (
    "You convert messy product review notes into structured JSON. "
    "Produce a JSON object with a top-level 'reviews' array. "
    "Each review must contain product_name, rating, reviewer, summary, and sentiment. "
    "Use null when information is missing. Sentiment must be positive, negative, or neutral. "
    "Before finishing, call submit_reviews with your structured payload. "
    "If validation fails, fix the data and try again. "
    "After validation succeeds, reply with only the final JSON object and no markdown."
)


def _extract_json_text(text: str) -> str:
    stripped = text.strip()
    if not stripped.startswith("```"):
        return stripped

    lines = stripped.splitlines()
    if lines and lines[0].startswith("```"):
        lines = lines[1:]
    if lines and lines[-1].startswith("```"):
        lines = lines[:-1]
    if lines and lines[0].strip().lower() == "json":
        lines = lines[1:]
    return "\n".join(lines).strip()


def parse_reviews_with_copilot(ctx, raw_review_data: str) -> dict:
    del ctx

    async def _parse_reviews_with_copilot() -> dict:
        client = CopilotClient()
        await client.start()
        try:
            session = await client.create_session({
                "model": "gpt-5.4",
                "on_permission_request": PermissionHandler.approve_all,
                "system_message": {"content": SYSTEM_PROMPT},
                "tools": [submit_reviews],
            })
            try:
                response = await session.send_and_wait({
                    "prompt": f"Parse and normalize these product reviews:\n\n{raw_review_data}",
                })
                content = response.data.content if response else ""
                if not content:
                    raise ValueError("Copilot returned no response.")

                parsed = json.loads(_extract_json_text(content))
                validated = ReviewList.model_validate(parsed)
                return validated.model_dump(mode="json")
            finally:
                await session.disconnect()
        finally:
            await client.stop()

    return asyncio.run(_parse_reviews_with_copilot())
