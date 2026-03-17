from __future__ import annotations

import asyncio
import json
import re
import sys
from pathlib import Path
from typing import Any

from durabletask import task

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from shared.copilot_activity import CopilotRequest, copilot_agent_activity

from agents import DECOMPOSER_AGENT, RESEARCHER_AGENT, SYNTHESIZER_AGENT

DEFAULT_MODEL = "gpt-5.4"


def _default_dimensions(query: str) -> list[str]:
    lowered = query.lower()
    if any(keyword in lowered for keyword in ("latency", "throughput", "performance")):
        return ["performance", "cost", "integration", "operational risk"]
    if any(keyword in lowered for keyword in ("agent", "assistant", "copilot")):
        return ["capabilities", "tooling", "governance", "developer experience"]
    return ["performance", "ecosystem", "operational fit", "cost"]


def _parse_dimensions(raw: str, query: str) -> list[str]:
    try:
        parsed = json.loads(raw)
        if isinstance(parsed, dict):
            parsed = parsed.get("dimensions", [])
        if isinstance(parsed, list):
            cleaned = [str(item).strip() for item in parsed if str(item).strip()]
            if cleaned:
                return cleaned[:5]
    except Exception:
        pass

    candidates = [
        part.strip(" -*0123456789.	")
        for part in raw.splitlines()
        if part.strip()
    ]
    cleaned = [item for item in candidates if item]
    return cleaned[:5] or _default_dimensions(query)


def _run_custom_agent(ctx: task.ActivityContext, prompt: str, agent_config: dict[str, Any]) -> str:
    async def _impl() -> str:
        request = CopilotRequest(
            prompt=prompt,
            model=DEFAULT_MODEL,
            custom_agents=[agent_config],
            agent=agent_config["name"],
        )
        response = await copilot_agent_activity(ctx, request)
        return response.content.strip()

    return asyncio.run(_impl())


def decompose_query(ctx: task.ActivityContext, input_: str) -> list[str]:
    prompt = (
        "Decompose this comparison question into 3-5 independent research dimensions. "
        "Return JSON as {\"dimensions\": [\"...\"]}.\n\n"
        f"Question: {input_}"
    )
    try:
        raw = _run_custom_agent(ctx, prompt, DECOMPOSER_AGENT)
        return _parse_dimensions(raw, input_)
    except Exception:
        return _default_dimensions(input_)


def research_dimension(ctx: task.ActivityContext, input_: dict[str, str]) -> str:
    query = input_["query"]
    dimension = input_["dimension"]
    prompt = (
        "Research one comparison dimension. Focus on factual trade-offs, meaningful differences, "
        "and practical implications. Provide a concise section with evidence-oriented bullets.\n\n"
        f"Question: {query}\n"
        f"Dimension: {dimension}"
    )
    try:
        raw = _run_custom_agent(ctx, prompt, RESEARCHER_AGENT)
        return raw or f"{dimension.title()}: No findings were returned."
    except Exception:
        summary = re.sub(r"\s+", " ", query).strip()
        return (
            f"{dimension.title()}: fallback findings for {summary}. "
            f"This dimension highlights the main trade-offs, implementation concerns, and areas needing validation."
        )


def synthesize_report(ctx: task.ActivityContext, input_: dict[str, Any]) -> str:
    query = input_["query"]
    findings = input_.get("findings", [])
    prompt = (
        "Create a structured comparison report with: executive summary, per-dimension analysis, "
        "and a final recommendation. Be crisp and actionable.\n\n"
        f"Question: {query}\n\n"
        "Findings:\n"
        + "\n\n".join(str(finding) for finding in findings)
    )
    try:
        raw = _run_custom_agent(ctx, prompt, SYNTHESIZER_AGENT)
        if raw:
            return raw
    except Exception:
        pass

    sections = ["# Deep Research Report", f"\nQuestion: {query}", "\n## Findings"]
    sections.extend(f"- {finding}" for finding in findings)
    sections.append("\n## Recommendation\nPrioritize the option that performs consistently across the strongest dimensions above.")
    return "\n".join(sections)
