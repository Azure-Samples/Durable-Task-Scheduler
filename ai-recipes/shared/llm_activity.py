"""Reusable LLM invocation activity for Durable Task AI Hub recipes.

This activity wraps the OpenAI API with Durable Task best practices:
- Client retries disabled (max_retries=0) so Durable Task manages retries
- Generic interface: model, instructions, input, and tools are all configurable
- Returns structured response with content and tool calls
"""

import json
import os
import durabletask.activity as activity
from openai import AsyncOpenAI
from shared.models import LlmRequest, LlmResponse


async def invoke_llm(ctx: activity.ActivityContext, request: LlmRequest) -> LlmResponse:
    """Invoke an LLM via OpenAI API. Retries are handled by Durable Task, not the OpenAI client."""
    # Disable client retries — Durable Task handles retry logic
    client = AsyncOpenAI(
        base_url=os.getenv("OPENAI_BASE_URL"),
        api_key=os.getenv("OPENAI_API_KEY"),
        max_retries=0,
    )

    kwargs: dict = {
        "model": request.model,
        "instructions": request.instructions,
        "input": request.input,
    }
    if request.tools:
        kwargs["tools"] = request.tools

    try:
        resp = await client.responses.create(**kwargs, timeout=60)

        # Parse response into tool calls and content
        tool_calls = []
        content = ""
        raw_output = []

        for item in resp.output:
            raw = item.model_dump() if hasattr(item, "model_dump") else item
            raw_output.append(raw)

            if item.type == "function_call":
                args = json.loads(item.arguments) if isinstance(item.arguments, str) else item.arguments
                tool_calls.append({
                    "name": item.name,
                    "arguments": args,
                    "call_id": item.call_id,
                })
            elif item.type == "message":
                for content_part in getattr(item, "content", []):
                    if hasattr(content_part, "text"):
                        content += content_part.text

        if not content and hasattr(resp, "output_text"):
            content = resp.output_text

        return LlmResponse(content=content, tool_calls=tool_calls, raw_output=raw_output)
    finally:
        await client.close()
