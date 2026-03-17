"""HTTP activities for GitHub REST API calls."""

from __future__ import annotations

import os
import time
from datetime import datetime, timezone
from typing import Any

import httpx
from durabletask import task

DEFAULT_USER_AGENT = os.getenv(
    "GITHUB_USER_AGENT",
    "durable-task-ai-hub/03-durable-mcp-tools (+https://github.com/microsoft/durabletask-python)",
)


def _build_headers() -> dict[str, str]:
    headers = {
        "Accept": "application/vnd.github+json",
        "User-Agent": DEFAULT_USER_AGENT,
        "X-GitHub-Api-Version": "2022-11-28",
    }
    token = os.getenv("GITHUB_TOKEN")
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return headers


def _format_rate_limit_message(url: str, response: httpx.Response) -> str:
    reset_raw = response.headers.get("x-ratelimit-reset")
    remaining = response.headers.get("x-ratelimit-remaining", "0")
    message = f"GitHub API rate limit reached while requesting {url}. Remaining requests: {remaining}."

    if reset_raw and reset_raw.isdigit():
        wait_seconds = max(0, int(reset_raw) - int(time.time()))
        reset_at = datetime.fromtimestamp(int(reset_raw), tz=timezone.utc)
        message += f" Limit resets at {reset_at.isoformat()} (about {wait_seconds} seconds)."

    if not os.getenv("GITHUB_TOKEN"):
        message += " Set GITHUB_TOKEN for a higher authenticated rate limit."

    return message


def fetch_github_api(ctx: task.ActivityContext, url: str) -> dict[str, Any]:
    """Perform an HTTP GET against the GitHub REST API with optional authentication."""
    timeout = httpx.Timeout(30.0, connect=10.0)

    try:
        with httpx.Client(headers=_build_headers(), timeout=timeout, follow_redirects=True) as client:
            response = client.get(url)
            if response.status_code in (403, 429) and response.headers.get("x-ratelimit-remaining") == "0":
                raise RuntimeError(_format_rate_limit_message(url, response))

            response.raise_for_status()
            return response.json()
    except httpx.HTTPStatusError as exc:
        details: Any
        try:
            details = exc.response.json()
        except ValueError:
            details = exc.response.text

        raise RuntimeError(
            f"GitHub request to {url} failed with status {exc.response.status_code}: {details}"
        ) from exc
    except httpx.RequestError as exc:
        raise RuntimeError(f"GitHub request to {url} failed: {exc}") from exc
