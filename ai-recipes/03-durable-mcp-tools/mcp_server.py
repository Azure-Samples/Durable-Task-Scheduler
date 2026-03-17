"""FastMCP server whose tools are backed by Durable Task orchestrations."""

from __future__ import annotations

import asyncio
import json
import os
from datetime import datetime, timezone
from types import MethodType
from typing import Any

from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.client import OrchestrationState, OrchestrationStatus
from fastmcp import FastMCP
from mcp import types as mcp_types

DEFAULT_ENDPOINT = "http://localhost:8080"
DEFAULT_TASKHUB = "default"
DEFAULT_TIMEOUT_SECONDS = int(os.getenv("DTS_TIMEOUT_SECONDS", "90"))
DEFAULT_TASK_TTL_MS = int(os.getenv("MCP_TASK_TTL_MS", "60000"))
TASK_POLL_INTERVAL_MS = int(os.getenv("MCP_TASK_POLL_INTERVAL_MS", "2000"))
TASK_CANCEL_REASON = "Cancelled via MCP tasks/cancel"

STATUS_MAP = {
    OrchestrationStatus.PENDING: "working",
    OrchestrationStatus.RUNNING: "working",
    OrchestrationStatus.SUSPENDED: "working",
    OrchestrationStatus.CONTINUED_AS_NEW: "working",
    OrchestrationStatus.COMPLETED: "completed",
    OrchestrationStatus.FAILED: "failed",
    OrchestrationStatus.TERMINATED: "cancelled",
}

REPO_INPUT_SCHEMA = {
    "type": "object",
    "properties": {
        "owner": {"type": "string", "description": "GitHub repository owner or organization."},
        "repo": {"type": "string", "description": "GitHub repository name."},
    },
    "required": ["owner", "repo"],
    "additionalProperties": False,
}

TASK_REGISTRY: dict[str, dict[str, Any]] = {}

mcp = FastMCP("github-inspector")


def get_durable_client() -> DurableTaskSchedulerClient:
    """Create a Durable Task client for the local DTS emulator."""
    endpoint = os.getenv("DTS_ENDPOINT", DEFAULT_ENDPOINT)
    taskhub = os.getenv("DTS_TASKHUB", DEFAULT_TASKHUB)
    secure_channel = endpoint.startswith("https://")

    return DurableTaskSchedulerClient(
        host_address=endpoint,
        taskhub=taskhub,
        token_credential=None,
        secure_channel=secure_channel,
    )


def _install_task_capabilities() -> None:
    """Advertise MCP Tasks support even though execution is backed by Durable Task."""
    original_get_capabilities = mcp._mcp_server.get_capabilities

    def patched_get_capabilities(self, notification_options, experimental_capabilities):
        capabilities = original_get_capabilities(notification_options, experimental_capabilities)
        capabilities.tasks = mcp_types.ServerTasksCapability(
            cancel=mcp_types.TasksCancelCapability(),
            requests=mcp_types.ServerTasksRequestsCapability(
                tools=mcp_types.TasksToolsCapability(call=mcp_types.TasksCallCapability())
            ),
        )
        return capabilities

    mcp._mcp_server.get_capabilities = MethodType(patched_get_capabilities, mcp._mcp_server)


def _normalize_repo_input(arguments: dict[str, Any] | None) -> dict[str, str]:
    owner = str((arguments or {}).get("owner", "")).strip()
    repo = str((arguments or {}).get("repo", "")).strip()

    if not owner or not repo:
        raise ValueError("Both owner and repo are required.")

    return {"owner": owner, "repo": repo}


def _task_ttl(task_meta: mcp_types.TaskMetadata | None) -> int | None:
    if task_meta and task_meta.ttl is not None:
        return task_meta.ttl
    return DEFAULT_TASK_TTL_MS


def _tool_result(text: str, *, is_error: bool = False) -> mcp_types.CallToolResult:
    return mcp_types.CallToolResult.model_validate(
        {
            "content": [{"type": "text", "text": text}],
            "isError": is_error,
        }
    )


def _tool_definition(name: str, description: str, *, task_support: str | None = None) -> mcp_types.Tool:
    payload: dict[str, Any] = {
        "name": name,
        "description": description,
        "inputSchema": REPO_INPUT_SCHEMA,
    }
    if task_support is not None:
        payload["execution"] = {"taskSupport": task_support}
    return mcp_types.Tool.model_validate(payload)


def _format_status_message(state: OrchestrationState) -> str | None:
    if state.serialized_custom_status:
        try:
            parsed = json.loads(state.serialized_custom_status)
        except json.JSONDecodeError:
            return state.serialized_custom_status
        if isinstance(parsed, str):
            return parsed
        return json.dumps(parsed)

    if state.runtime_status == OrchestrationStatus.FAILED and state.failure_details is not None:
        return str(state.failure_details)

    if state.runtime_status == OrchestrationStatus.TERMINATED:
        return TASK_CANCEL_REASON

    return None


def _task_payload(task_id: str, state: OrchestrationState | None = None) -> dict[str, Any]:
    now = datetime.now(timezone.utc)
    metadata = TASK_REGISTRY.get(task_id, {})

    created_at = state.created_at if state is not None else metadata.get("created_at", now)
    last_updated_at = state.last_updated_at if state is not None else metadata.get("last_updated_at", created_at)
    ttl = metadata.get("ttl", DEFAULT_TASK_TTL_MS)
    status = STATUS_MAP.get(state.runtime_status, "working") if state is not None else "working"
    status_message = _format_status_message(state) if state is not None else None

    return {
        "taskId": task_id,
        "status": status,
        "statusMessage": status_message,
        "createdAt": created_at,
        "lastUpdatedAt": last_updated_at,
        "ttl": ttl,
        "pollInterval": TASK_POLL_INTERVAL_MS,
    }


async def get_task_status(client: DurableTaskSchedulerClient, task_id: str) -> dict[str, Any]:
    """Map Durable Task orchestration state to MCP task status."""
    state = await asyncio.to_thread(client.get_orchestration_state, task_id)
    if state is None:
        raise ValueError(f"Unknown task ID: {task_id}")
    return _task_payload(task_id, state)


async def schedule_workflow(orchestration_name: str, input_value: Any) -> str:
    """Start an orchestration and return the instance ID."""
    client = get_durable_client()
    return await asyncio.to_thread(
        client.schedule_new_orchestration,
        orchestration_name,
        input=input_value,
    )


async def run_workflow(orchestration_name: str, input_value: Any) -> str:
    """Start an orchestration and wait for its durable result."""
    client = get_durable_client()

    instance_id = await asyncio.to_thread(
        client.schedule_new_orchestration,
        orchestration_name,
        input=input_value,
    )
    state = await asyncio.to_thread(
        client.wait_for_orchestration_completion,
        instance_id,
        fetch_payloads=True,
        timeout=DEFAULT_TIMEOUT_SECONDS,
    )

    if state is None:
        raise TimeoutError(f"Orchestration {orchestration_name!r} did not complete before the timeout.")

    state.raise_if_failed()
    return state.serialized_output or ""


async def _handle_sync_tool(name: str, arguments: dict[str, Any] | None) -> mcp_types.CallToolResult:
    repo_ref = _normalize_repo_input(arguments)

    if name == "inspect_repo":
        result = await run_workflow("GetRepoInfo", repo_ref)
        return _tool_result(result)

    if name == "recent_activity":
        result = await run_workflow("GetRecentActivity", repo_ref)
        return _tool_result(result)

    return _tool_result(f"Unknown tool: {name}", is_error=True)


async def _handle_task_tool(
    name: str,
    arguments: dict[str, Any] | None,
    task_meta: mcp_types.TaskMetadata | None,
) -> mcp_types.CreateTaskResult:
    if name != "recent_activity":
        raise ValueError(f"Tool {name!r} does not support task execution.")

    repo_ref = _normalize_repo_input(arguments)
    instance_id = await schedule_workflow("GetRecentActivity", repo_ref)
    created_at = datetime.now(timezone.utc)
    ttl = _task_ttl(task_meta)

    TASK_REGISTRY[instance_id] = {
        "tool_name": name,
        "ttl": ttl,
        "created_at": created_at,
        "arguments": repo_ref,
    }

    return mcp_types.CreateTaskResult.model_validate(
        {
            "task": {
                "taskId": instance_id,
                "status": "working",
                "createdAt": created_at,
                "lastUpdatedAt": created_at,
                "ttl": ttl,
                "pollInterval": TASK_POLL_INTERVAL_MS,
            }
        }
    )


async def _list_tools(_ctx: Any, _params: Any) -> mcp_types.ListToolsResult:
    return mcp_types.ListToolsResult.model_validate(
        {
            "tools": [
                _tool_definition("inspect_repo", "Get detailed information about a GitHub repository."),
                _tool_definition(
                    "recent_activity",
                    "Get recent activity (commits, issues, PRs) for a GitHub repository.",
                    task_support="optional",
                ),
            ]
        }
    )


async def _call_tool(_ctx: Any, params: mcp_types.CallToolRequestParams) -> mcp_types.CallToolResult | mcp_types.CreateTaskResult:
    if params.task is not None:
        return await _handle_task_tool(params.name, params.arguments, params.task)
    return await _handle_sync_tool(params.name, params.arguments)


async def _get_task(_ctx: Any, params: mcp_types.GetTaskRequestParams) -> mcp_types.GetTaskResult:
    status = await get_task_status(get_durable_client(), params.task_id)
    return mcp_types.GetTaskResult.model_validate(status)


async def _get_task_result(_ctx: Any, params: mcp_types.GetTaskPayloadRequestParams) -> mcp_types.GetTaskPayloadResult:
    client = get_durable_client()
    state = await asyncio.to_thread(
        client.wait_for_orchestration_completion,
        params.task_id,
        fetch_payloads=True,
        timeout=DEFAULT_TIMEOUT_SECONDS,
    )

    if state is None:
        raise TimeoutError(f"Task {params.task_id!r} did not finish before the timeout.")

    if state.runtime_status == OrchestrationStatus.FAILED:
        message = _format_status_message(state) or f"Task {params.task_id!r} failed."
        raise RuntimeError(message)

    if state.runtime_status == OrchestrationStatus.TERMINATED:
        raise RuntimeError(TASK_CANCEL_REASON)

    result = _tool_result(state.serialized_output or "")
    return mcp_types.GetTaskPayloadResult.model_validate(result.model_dump(by_alias=True, exclude_none=True))


async def _cancel_task(_ctx: Any, params: mcp_types.CancelTaskRequestParams) -> mcp_types.CancelTaskResult:
    client = get_durable_client()
    await asyncio.to_thread(client.terminate_orchestration, params.task_id, TASK_CANCEL_REASON)

    state = await asyncio.to_thread(client.get_orchestration_state, params.task_id)
    payload = _task_payload(params.task_id, state)
    payload["status"] = "cancelled"
    payload["statusMessage"] = TASK_CANCEL_REASON
    payload["lastUpdatedAt"] = datetime.now(timezone.utc)
    return mcp_types.CancelTaskResult.model_validate(payload)


_install_task_capabilities()
mcp._mcp_server._request_handlers["tools/list"] = _list_tools
mcp._mcp_server._request_handlers["tools/call"] = _call_tool
mcp._mcp_server._request_handlers["tasks/get"] = _get_task
mcp._mcp_server._request_handlers["tasks/result"] = _get_task_result
mcp._mcp_server._request_handlers["tasks/cancel"] = _cancel_task


if __name__ == "__main__":
    mcp.run()
