from __future__ import annotations

import asyncio
from datetime import UTC, datetime
from textwrap import dedent
from typing import Any

from copilot import CopilotClient, PermissionHandler
from copilot.tools import define_tool
from durabletask import task
from pydantic import BaseModel, Field


class RepoInspectionParams(BaseModel):
    repo: str = Field(description="Repository name to inspect, such as microsoft/Durable-Task-Scheduler")


@define_tool(description="Get a simulated snapshot of recent commits for a repository.")
def get_recent_commits(params: RepoInspectionParams) -> str:
    repo = params.repo.strip() or "unknown-repo"
    return dedent(
        f"""\
        Recent commits for {repo}:
        - a1c9f42 — Tightened retry handling for scheduled review fan-out.
        - b37de10 — Added durable continue-as-new checkpoint logging.
        - c8427aa — Refined worker startup messaging for local emulator runs.
        """
    ).strip()


@define_tool(description="Get a simulated snapshot of open issues for a repository.")
def get_open_issues(params: RepoInspectionParams) -> str:
    repo = params.repo.strip() or "unknown-repo"
    return dedent(
        f"""\
        Open issues for {repo}:
        - #214: Scheduled summaries should include deployment drift indicators [enhancement, monitoring]
        - #219: Activity retries can hide flaky downstream tool failures [bug, reliability]
        - #223: Add richer examples for Copilot SDK custom agents [documentation, good first issue]
        """
    ).strip()


@define_tool(description="Get a simulated snapshot of recent pull request activity for a repository.")
def get_pr_activity(params: RepoInspectionParams) -> str:
    repo = params.repo.strip() or "unknown-repo"
    return dedent(
        f"""\
        Recent pull request activity for {repo}:
        - PR #301 merged — Added structured status payloads for background orchestration runs.
        - PR #305 open — Improve MCP task agent fallback messaging before approval.
        - PR #308 review requested — Expand Copilot SDK recipe coverage for tool-enabled agents.
        """
    ).strip()


CODEBASE_MONITOR_AGENT = {
    "name": "codebase-monitor",
    "display_name": "Codebase Monitor",
    "description": "Specialized in inspecting simulated repository activity before writing scheduled health summaries.",
    "prompt": (
        "You are the Codebase Monitor. Before producing any summary, use the available inspection tools "
        "to gather fresh commit, issue, and pull request data for the target repository. Ground your report "
        "in the tool results, then provide a concise scheduled update with highlights, active risks, and "
        "recommended follow-up actions. Do not answer from general knowledge when tool data is available."
    ),
}


def run_scheduled_review(_ctx: task.ActivityContext, config: dict[str, Any]) -> str:
    """Run a Copilot SDK codebase monitoring agent for a scheduled task."""

    async def _impl() -> str:
        client = CopilotClient()
        await client.start()
        try:
            session = await client.create_session(
                {
                    "model": "gpt-5.4",
                    "on_permission_request": PermissionHandler.approve_all,
                    "tools": [get_recent_commits, get_open_issues, get_pr_activity],
                    "custom_agents": [CODEBASE_MONITOR_AGENT],
                    "agent": CODEBASE_MONITOR_AGENT["name"],
                }
            )
            try:
                prompt = config.get("prompt", "Summarize recent changes in the codebase")
                repo = str(config.get("repo") or config.get("repository") or "microsoft/Durable-Task-Scheduler")
                timestamp = datetime.now(UTC).isoformat()
                full_prompt = dedent(
                    f"""\
                    [Scheduled run at {timestamp}]
                    Repository: {repo}
                    Task: {prompt}

                    Use the inspection tools before summarizing so the report reflects fresh repository activity.
                    """
                ).strip()
                response = await session.send_and_wait({"prompt": full_prompt})
                if response and getattr(response, "data", None):
                    return getattr(response.data, "content", None) or "No response"
                return "No response"
            finally:
                await session.disconnect()
        finally:
            await client.stop()

    return asyncio.run(_impl())


def store_report(_ctx: task.ActivityContext, report: dict[str, Any]) -> str:
    """Store the agent report (simulated)."""
    print(f"[{report['timestamp']}] Report stored: {report['content'][:100]}...")
    return "stored"
