"""Durable orchestrations that back the MCP GitHub inspector tools."""

from __future__ import annotations

from datetime import timedelta
from typing import Any
from urllib.parse import quote_plus

from durabletask import task
from durabletask.task import RetryPolicy

GITHUB_RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=2),
    max_number_of_attempts=4,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=20),
    retry_timeout=timedelta(minutes=2),
)


def _normalize_repo_reference(input_: dict[str, Any] | str) -> tuple[str, str]:
    if isinstance(input_, str):
        cleaned = input_.strip().strip("/")
        if "/" not in cleaned:
            raise ValueError("Repository input must look like 'owner/repo'.")
        owner, repo = cleaned.split("/", 1)
    else:
        owner = str(input_.get("owner", "")).strip()
        repo = str(input_.get("repo", "")).strip()

    if not owner or not repo:
        raise ValueError("Both owner and repo are required.")

    return owner, repo


def _format_repo_info(owner: str, repo: str, response: dict[str, Any]) -> str:
    license_name = response.get("license", {}).get("spdx_id") or response.get("license", {}).get("name") or "Not specified"
    topics = response.get("topics", [])
    lines = [
        f"Repository: {response.get('full_name', f'{owner}/{repo}')}",
        f"Description: {response.get('description') or 'No description provided.'}",
        f"Primary language: {response.get('language') or 'Not specified'}",
        f"License: {license_name}",
        (
            "Stars: "
            f"{response.get('stargazers_count', 0):,} | Forks: {response.get('forks_count', 0):,} | "
            f"Watchers: {response.get('subscribers_count', 0):,}"
        ),
        (
            "Open issues: "
            f"{response.get('open_issues_count', 0):,} | Default branch: {response.get('default_branch', 'main')} | "
            f"Visibility: {response.get('visibility', 'public')}"
        ),
        f"Last pushed: {response.get('pushed_at') or 'Unknown'}",
        f"URL: {response.get('html_url', f'https://github.com/{owner}/{repo}')}",
    ]

    if topics:
        lines.append("Topics: " + ", ".join(topics[:8]))
    if response.get("homepage"):
        lines.append(f"Homepage: {response['homepage']}")

    return "\n".join(lines)


def _search_url(kind: str, query: str, *, sort: str) -> str:
    return (
        f"https://api.github.com/search/{kind}?q={quote_plus(query)}"
        f"&sort={quote_plus(sort)}&order=desc&per_page=5"
    )


def _format_recent_activity(
    owner: str,
    repo: str,
    commits_response: dict[str, Any],
    issues_response: dict[str, Any],
    pulls_response: dict[str, Any],
) -> str:
    commits = commits_response.get("items", [])
    issues = issues_response.get("items", [])
    pulls = pulls_response.get("items", [])

    lines = [f"Recent GitHub activity for {owner}/{repo}:", "", "Recent commits:"]
    if commits:
        for commit in commits[:5]:
            sha = str(commit.get("sha", ""))[:7] or "unknown"
            message = commit.get("commit", {}).get("message", "No commit message.").splitlines()[0]
            author = commit.get("author", {}).get("login") or commit.get("commit", {}).get("author", {}).get("name") or "unknown"
            lines.append(f"- {sha} by {author}: {message}")
    else:
        lines.append("- No recent commits found.")

    lines.extend(["", "Recent issues:"])
    if issues:
        for issue in issues[:5]:
            lines.append(
                f"- #{issue.get('number')} [{issue.get('state', 'open')}] {issue.get('title', 'Untitled issue')} "
                f"(updated {issue.get('updated_at', 'unknown')})"
            )
    else:
        lines.append("- No recent issues found.")

    lines.extend(["", "Recent pull requests:"])
    if pulls:
        for pull in pulls[:5]:
            lines.append(
                f"- #{pull.get('number')} [{pull.get('state', 'open')}] {pull.get('title', 'Untitled PR')} "
                f"(updated {pull.get('updated_at', 'unknown')})"
            )
    else:
        lines.append("- No recent pull requests found.")

    return "\n".join(lines)


def GetRepoInfo(ctx: task.OrchestrationContext, repo_ref: dict[str, Any] | str):
    """Fetch repository metadata and format it for MCP clients."""
    owner, repo = _normalize_repo_reference(repo_ref)
    repo_url = f"https://api.github.com/repos/{owner}/{repo}"
    response = yield ctx.call_activity(
        "fetch_github_api",
        input=repo_url,
        retry_policy=GITHUB_RETRY_POLICY,
    )
    return _format_repo_info(owner, repo, response)


def GetRecentActivity(ctx: task.OrchestrationContext, repo_ref: dict[str, Any] | str):
    """Fetch recent commits, issues, and pull requests for a repository."""
    owner, repo = _normalize_repo_reference(repo_ref)
    repo_name = f"{owner}/{repo}"

    commits_task = ctx.call_activity(
        "fetch_github_api",
        input=_search_url("commits", f"repo:{repo_name}", sort="author-date"),
        retry_policy=GITHUB_RETRY_POLICY,
    )
    issues_task = ctx.call_activity(
        "fetch_github_api",
        input=_search_url("issues", f"repo:{repo_name} type:issue", sort="updated"),
        retry_policy=GITHUB_RETRY_POLICY,
    )
    pulls_task = ctx.call_activity(
        "fetch_github_api",
        input=_search_url("issues", f"repo:{repo_name} type:pr", sort="updated"),
        retry_policy=GITHUB_RETRY_POLICY,
    )

    commits_response, issues_response, pulls_response = yield task.when_all(
        [commits_task, issues_task, pulls_task]
    )
    return _format_recent_activity(owner, repo, commits_response, issues_response, pulls_response)
