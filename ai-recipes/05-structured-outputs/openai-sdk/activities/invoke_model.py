from __future__ import annotations

import os
from pathlib import Path
from typing import Any

from dotenv import load_dotenv
from durabletask import task
from openai import OpenAI
from pydantic import BaseModel

from models.reviews import ProductReviewList

load_dotenv(Path(__file__).resolve().parents[3] / ".env")


FORMAT_REGISTRY: dict[str, type[BaseModel]] = {
    "ProductReviewList": ProductReviewList,
}

SYSTEM_PROMPT = """Extract normalized product review data and return only valid structured output.
- Use null when a field is missing or ambiguous.
- Ratings must be integers from 1 to 5.
- Choose sentiment from positive, mixed, or negative.
- Keep summaries short and factual.
"""


def _extract_parsed_model(response: Any) -> BaseModel:
    for item in getattr(response, "output", []):
        for content in getattr(item, "content", []):
            parsed = getattr(content, "parsed", None)
            if parsed is not None:
                return parsed
    raise ValueError("The model response did not include parsed structured output.")



def invoke_model(ctx: task.ActivityContext, request: dict) -> dict:
    """Call OpenAI with structured outputs and return a validated payload."""
    schema_name = request["schema_name"]
    input_text = request["input_text"]

    response_model = FORMAT_REGISTRY[schema_name]
    client = OpenAI(
        base_url=os.getenv("OPENAI_BASE_URL"),
        api_key=os.getenv("OPENAI_API_KEY"),
        max_retries=0,
    )
    response = client.responses.parse(
        model=os.getenv("OPENAI_MODEL", "gpt-5.4"),
        input=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {
                "role": "user",
                "content": (
                    "Transform these messy product review notes into the requested schema.\n\n"
                    f"Review feed:\n{input_text}"
                ),
            },
        ],
        text_format=response_model,
    )

    parsed_model = _extract_parsed_model(response)
    return parsed_model.model_dump(mode="json")
