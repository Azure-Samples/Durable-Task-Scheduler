from __future__ import annotations

import hashlib
import json
import re
from datetime import timedelta
from typing import Any

from durabletask import task
from durabletask.entities import DurableEntity, EntityInstanceId
from durabletask.task import RetryPolicy

from activities import run_executor_agent, run_planner_agent, run_reviewer_agent

AGENT_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=3),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


class sharedstate(DurableEntity):
    def _load_state(self) -> dict[str, Any]:
        return self.get_state(dict, {"plan": None, "results": [], "status": {}})

    def set_plan(self, plan: str) -> dict[str, Any]:
        state = self._load_state()
        state["plan"] = plan
        self.set_state(state)
        return state

    def add_result(self, result: dict[str, Any]) -> dict[str, Any]:
        state = self._load_state()
        state["results"].append(result)
        self.set_state(state)
        return state

    def set_status(self, status_update: dict[str, str]) -> dict[str, str]:
        state = self._load_state()
        state["status"][status_update.get("agent", "unknown")] = status_update.get("status", "unknown")
        self.set_state(state)
        return state["status"]

    def snapshot(self, _input=None) -> dict[str, Any]:
        return self._load_state()


SharedStateEntity = sharedstate


def _slugify(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-") or "task"


def _executor_key(sub_task: str) -> str:
    slug = _slugify(sub_task)
    digest = hashlib.sha1(slug.encode("utf-8")).hexdigest()[:10]
    trimmed = slug[:32].strip("-") or "task"
    return f"executor-{trimmed}-{digest}"


def parse_plan(plan: str) -> list[str]:
    try:
        parsed = json.loads(plan)
        if isinstance(parsed, dict):
            items = parsed.get("sub_tasks") or parsed.get("tasks") or []
        elif isinstance(parsed, list):
            items = parsed
        else:
            items = []
        cleaned = [str(item).strip() for item in items if str(item).strip()]
        if cleaned:
            return cleaned[:3]
    except Exception:
        pass

    fallback = [line.strip(" -*0123456789.	") for line in plan.splitlines() if line.strip()]
    return fallback[:3] or ["Clarify requirements", "Execute implementation", "Review results"]


def executor_session(ctx: task.OrchestrationContext, input_: dict[str, str]):
    sub_task = input_["sub_task"]
    shared_key = input_["shared_entity_key"]
    agent_name = _executor_key(sub_task)
    entity_id = EntityInstanceId("sharedstate", shared_key)

    ctx.signal_entity(entity_id, "set_status", {"agent": agent_name, "status": "running"})
    result = yield ctx.call_activity(run_executor_agent, input=sub_task, retry_policy=AGENT_RETRY)
    payload = {"sub_task": sub_task, "result": result}
    ctx.signal_entity(entity_id, "add_result", payload)
    ctx.signal_entity(entity_id, "set_status", {"agent": agent_name, "status": "completed"})
    return payload


def multi_agent_orchestration(ctx: task.OrchestrationContext, task_description: str):
    entity_id = EntityInstanceId("sharedstate", ctx.instance_id)
    ctx.signal_entity(entity_id, "set_status", {"agent": "planner", "status": "running"})
    ctx.set_custom_status({"stage": "planning", "task": task_description})

    plan = yield ctx.call_activity(run_planner_agent, input=task_description, retry_policy=AGENT_RETRY)
    sub_tasks = parse_plan(plan)
    ctx.signal_entity(entity_id, "set_plan", plan)
    ctx.signal_entity(entity_id, "set_status", {"agent": "planner", "status": "completed"})

    ctx.set_custom_status({"stage": "executing", "sub_task_count": len(sub_tasks)})
    executor_tasks = [
        (
            sub_task,
            _executor_key(sub_task),
        )
        for sub_task in sub_tasks
    ]
    results = yield task.when_all(
        [
        ctx.call_sub_orchestrator(
            executor_session,
            input={"sub_task": sub_task, "shared_entity_key": ctx.instance_id},
            instance_id=f"{ctx.instance_id}:{executor_key}",
            retry_policy=AGENT_RETRY,
        )
            for sub_task, executor_key in executor_tasks
        ]
    )

    ctx.signal_entity(entity_id, "set_status", {"agent": "reviewer", "status": "running"})
    snapshot = yield ctx.call_entity(entity_id, "snapshot")
    review_input = json.dumps(
        {
            "task_description": task_description,
            "plan": plan,
            "results": results,
            "shared_state": snapshot,
        },
        sort_keys=True,
    )
    review = yield ctx.call_activity(run_reviewer_agent, input=review_input, retry_policy=AGENT_RETRY)
    ctx.signal_entity(entity_id, "set_status", {"agent": "reviewer", "status": "completed"})
    return review
