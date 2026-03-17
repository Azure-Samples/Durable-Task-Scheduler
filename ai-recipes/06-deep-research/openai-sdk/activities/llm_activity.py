from __future__ import annotations

import json
import os
import re
from pathlib import Path
from typing import Any, Union

from dotenv import load_dotenv
from durabletask import task
from openai import OpenAI
from pydantic import BaseModel, Field

load_dotenv(Path(__file__).resolve().parents[3] / ".env")


class LLMRequest(BaseModel):
    system_prompt: str = Field(default="You are a helpful competitive analysis assistant.")
    user_prompt: str
    model: str = Field(default="gpt-5.4")
    temperature: float = Field(default=0.2)


def _extract_question(prompt: str) -> str:
    for marker in ("Comparison query:", "Research question:", "Original query:", "Task:", "Question:"):
        if marker in prompt:
            return prompt.split(marker, 1)[1].strip().splitlines()[0]
    return prompt.strip().splitlines()[-1]


def _extract_field(prompt: str, label: str) -> str:
    match = re.search(rf"{re.escape(label)}:\s*(.+)", prompt)
    return match.group(1).strip() if match else ""


def _extract_products(query: str) -> list[str]:
    match = re.search(r"compare\s+(.+?)(?:\s+for\s+|\s*$)", query, flags=re.IGNORECASE)
    candidate = match.group(1) if match else query
    parts = [part.strip(" .") for part in re.split(r"\bvs\.?\b|\bversus\b|,", candidate, flags=re.IGNORECASE) if part.strip(" .")]

    unique_products: list[str] = []
    for part in parts:
        if part not in unique_products:
            unique_products.append(part)

    return unique_products[:4] or ["Option A", "Option B"]


def _default_dimensions(query: str) -> list[str]:
    lowered = query.lower()
    if any(keyword in lowered for keyword in ("database", "postgresql", "mysql", "sqlite")):
        return ["performance", "operational complexity", "ecosystem", "portability"]
    if any(keyword in lowered for keyword in ("react", "vue", "svelte", "frontend")):
        return ["ecosystem", "learning curve", "performance", "enterprise fit"]
    return ["performance", "ecosystem", "learning curve", "operational fit"]


def _mock_response(request: LLMRequest) -> str:
    prompt = f"{request.system_prompt}\n{request.user_prompt}"
    lowered = prompt.lower()
    subject = _extract_question(request.user_prompt)

    if "products" in lowered and "dimensions" in lowered and "comparison" in lowered:
        return json.dumps(
            {
                "products": _extract_products(subject),
                "dimensions": _default_dimensions(subject),
            }
        )

    if "search_query" in lowered and "focus" in lowered and "done" in lowered and "dimension:" in lowered:
        iteration_match = re.search(r"Iteration:\s*(\d+)", prompt)
        iteration = int(iteration_match.group(1)) if iteration_match else 1
        dimension = _extract_field(prompt, "Dimension") or "adoption"
        focus_options = {
            "performance": ["benchmarks and throughput", "latency and scaling ceilings"],
            "ecosystem": ["libraries and community depth", "support and hiring signal"],
            "learning curve": ["team ramp-up", "day-two productivity"],
            "enterprise fit": ["governance and predictability", "talent availability"],
            "operational complexity": ["deployment overhead", "backup and observability"],
            "portability": ["deployment flexibility", "vendor lock-in risk"],
            "operational fit": ["deployment model", "maintenance burden"],
        }
        options = focus_options.get(dimension.lower(), ["core trade-offs", "decision risks"])
        focus = options[min(iteration - 1, len(options) - 1)]
        return json.dumps(
            {
                "search_query": f"{subject} {dimension} {focus}",
                "focus": focus,
                "done": iteration >= 2,
                "reason": f"Mock analyst is collecting {dimension} evidence about {focus}.",
            }
        )

    if "matrix" in lowered or "recommendation" in lowered:
        return (
            f"Mock comparison synthesis for {subject}:\n"
            "- Organize the evidence into a decision-ready matrix.\n"
            "- Highlight where each option is strongest and where it creates operational trade-offs.\n"
            "- End with a recommendation tied to the stated scenario."
        )

    return f"Mock LLM response for: {subject}"


def llm_activity(ctx: task.ActivityContext, input_: Union[dict[str, Any], LLMRequest]) -> str:
    request = input_ if isinstance(input_, LLMRequest) else LLMRequest.model_validate(input_)
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        return _mock_response(request)

    client = OpenAI(api_key=os.getenv("OPENAI_API_KEY"), base_url=os.getenv("OPENAI_BASE_URL"), max_retries=0)
    response = client.chat.completions.create(
        model=request.model,
        temperature=request.temperature,
        messages=[
            {"role": "system", "content": request.system_prompt},
            {"role": "user", "content": request.user_prompt},
        ],
    )
    content = response.choices[0].message.content
    return content.strip() if content else ""
