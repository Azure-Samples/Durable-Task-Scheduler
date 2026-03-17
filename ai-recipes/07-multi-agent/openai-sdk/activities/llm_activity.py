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
    system_prompt: str = Field(default="You are a multi-agent coordination assistant.")
    user_prompt: str
    model: str = Field(default="gpt-5.4")
    temperature: float = Field(default=0.2)


def _extract_task(prompt: str) -> str:
    for marker in ("Complex task:", "Task:", "User objective:"):
        if marker in prompt:
            return prompt.split(marker, 1)[1].strip().splitlines()[0]
    return prompt.strip().splitlines()[-1]


def _mock_response(request: LLMRequest) -> str:
    prompt = f"{request.system_prompt}\n{request.user_prompt}"
    lowered = prompt.lower()
    subject = _extract_task(request.user_prompt)

    if "agent assignments" in lowered or "decompose" in lowered:
        return json.dumps(
            {
                "agents": [
                    {
                        "role": "Planner",
                        "goal": f"Break down {subject} into a concrete execution plan.",
                        "tools": ["plan", "search"],
                    },
                    {
                        "role": "Researcher",
                        "goal": f"Collect evidence and examples relevant to {subject}.",
                        "tools": ["search", "summarize"],
                    },
                    {
                        "role": "Critic",
                        "goal": f"Identify gaps, risks, and trade-offs for {subject}.",
                        "tools": ["analyze", "summarize"],
                    },
                ]
            }
        )

    if "tool_name" in lowered and "tool_input" in lowered:
        match = re.search(r"Iteration:\s*(\d+)", prompt)
        iteration = int(match.group(1)) if match else 1
        role_match = re.search(r"Role:\s*(.+)", request.user_prompt)
        role = role_match.group(1).strip() if role_match else "Agent"
        tool_name = "search" if iteration == 1 else "summarize"
        return json.dumps(
            {
                "tool_name": tool_name,
                "tool_input": f"{role} perspective on {subject}",
                "done": iteration >= 2,
            }
        )

    if "final synthesis" in lowered or "combined result" in lowered:
        return (
            f"Final multi-agent synthesis for {subject}:\n\n"
            "- Planner outlined a coordinated execution approach.\n"
            "- Researcher contributed supporting evidence and examples.\n"
            "- Critic surfaced gaps, trade-offs, and operational risks.\n"
            "- Shared durable state kept the collaboration observable and recoverable."
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
