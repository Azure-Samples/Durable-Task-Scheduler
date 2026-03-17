from __future__ import annotations

import os
from pathlib import Path
from typing import Any, Optional

from dotenv import load_dotenv
from openai import OpenAI
from pydantic import BaseModel, Field

load_dotenv(Path(__file__).resolve().parents[3] / ".env")


class LlmActivityInput(BaseModel):
    model: str
    instructions: Optional[str] = None
    input: list[dict[str, Any]]
    tools: list[dict[str, Any]] = Field(default_factory=list)


def _normalize_output_item(item: Any) -> dict[str, Any]:
    if isinstance(item, dict):
        return item
    if hasattr(item, 'model_dump'):
        return item.model_dump()
    raise TypeError(f'Unsupported response output item: {type(item)!r}')


def invoke_llm(ctx: Any, payload: dict[str, Any]) -> dict[str, Any]:
    request = LlmActivityInput.model_validate(payload)
    client = OpenAI(
        base_url=os.getenv("OPENAI_BASE_URL"),
        api_key=os.getenv("OPENAI_API_KEY"),
        max_retries=0,
    )

    response = client.responses.create(
        model=request.model,
        instructions=request.instructions,
        input=request.input,
        tools=request.tools,
    )

    return {
        'id': response.id,
        'output_text': response.output_text,
        'output': [_normalize_output_item(item) for item in response.output],
    }
