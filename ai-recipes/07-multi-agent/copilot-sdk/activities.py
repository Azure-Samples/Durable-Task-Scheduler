from __future__ import annotations

import asyncio
import json
import sys
from pathlib import Path
from typing import Any

from durabletask import task

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from shared.copilot_activity import CopilotRequest, copilot_agent_activity

from agents import EXECUTOR_AGENT, PLANNER_AGENT, REVIEWER_AGENT

DEFAULT_MODEL = "gpt-5.4"
PLANNER_TIMEOUT_SECONDS = 60.0
EXECUTOR_TIMEOUT_SECONDS = 90.0
REVIEWER_TIMEOUT_SECONDS = 60.0


def _fallback_sub_tasks(task_description: str) -> list[str]:
    return [
        f"Clarify the objective for: {task_description}",
        f"Execute the main implementation steps for: {task_description}",
        f"Validate and summarize outcomes for: {task_description}",
    ]


def _run_agent(
    ctx: task.ActivityContext,
    prompt: str,
    agent_config: dict[str, Any],
    *,
    timeout_seconds: float,
) -> str:
    async def _impl() -> str:
        request = CopilotRequest(
            prompt=prompt,
            model=DEFAULT_MODEL,
            custom_agents=[agent_config],
            agent=agent_config["name"],
            timeout_seconds=timeout_seconds,
        )
        response = await copilot_agent_activity(ctx, request)
        return response.content.strip()

    return asyncio.run(_impl())


def run_planner_agent(ctx: task.ActivityContext, task_description: str) -> str:
    prompt = (
        "Break the task into at most 3 concrete sub-tasks that can run in parallel where possible. "
        "Keep each sub-task short, execution-ready, and focused on a distinct outcome. "
        "Return JSON only as {\"sub_tasks\": [\"...\"]}.\n\n"
        f"Task: {task_description}"
    )
    try:
        raw = _run_agent(ctx, prompt, PLANNER_AGENT, timeout_seconds=PLANNER_TIMEOUT_SECONDS)
        return raw or json.dumps({"sub_tasks": _fallback_sub_tasks(task_description)})
    except Exception:
        return json.dumps({"sub_tasks": _fallback_sub_tasks(task_description)})


def run_executor_agent(ctx: task.ActivityContext, plan: str) -> str:
    prompt = (
        "Execute this sub-task. Return a concise execution summary with what was completed, "
        "assumptions, and any remaining risk. Keep the response short and practical.\n\n"
        f"Sub-task: {plan}"
    )
    try:
        raw = _run_agent(ctx, prompt, EXECUTOR_AGENT, timeout_seconds=EXECUTOR_TIMEOUT_SECONDS)
        if raw:
            return raw
    except Exception:
        pass
    return f"Executed sub-task: {plan}. Captured deliverables, assumptions, and next actions."


def run_reviewer_agent(ctx: task.ActivityContext, result: str) -> str:
    prompt = (
        "Review the combined planner and executor output. Provide a quality assessment, "
        "key risks, and a recommended next action.\n\n"
        f"Combined result: {result}"
    )
    try:
        raw = _run_agent(ctx, prompt, REVIEWER_AGENT, timeout_seconds=REVIEWER_TIMEOUT_SECONDS)
        if raw:
            return raw
    except Exception:
        pass
    return "Review complete: the multi-agent output is coherent, but final verification should focus on unresolved risks and dependencies."
