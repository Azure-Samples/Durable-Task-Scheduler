from __future__ import annotations

import json
from datetime import timedelta

from durabletask import task
from durabletask.entities import EntityInstanceId
from durabletask.task import RetryPolicy
from pydantic import BaseModel, Field

from activities.llm_activity import llm_activity
from orchestrations.agent import agent


LLM_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


class AgentAssignment(BaseModel):
    role: str
    goal: str
    tools: list[str] = Field(default_factory=list)


class AssignmentPlan(BaseModel):
    agents: list[AgentAssignment] = Field(default_factory=list)


def _parse_assignments(raw: str, complex_task: str) -> list[AgentAssignment]:
    try:
        plan = AssignmentPlan.model_validate(json.loads(raw))
        if plan.agents:
            return plan.agents[:5]
    except Exception:
        pass
    return [
        AgentAssignment(role="Planner", goal=f"Break down {complex_task}", tools=["plan", "search"]),
        AgentAssignment(role="Researcher", goal=f"Gather evidence for {complex_task}", tools=["search", "summarize"]),
        AgentAssignment(role="Critic", goal=f"Surface trade-offs for {complex_task}", tools=["analyze", "summarize"]),
    ]


def coordinator(ctx: task.OrchestrationContext, complex_task: str):
    entity_id = EntityInstanceId("sharedstate", ctx.instance_id)
    ctx.signal_entity(entity_id, "set_status", {"agent": "coordinator", "status": "planning"})
    plan_prompt = {
        "system_prompt": (
            "Decompose the user objective into agent assignments. "
            "Return JSON with an agents array. Each agent should contain role, goal, and tools."
        ),
        "user_prompt": f"Complex task: {complex_task}",
        "temperature": 0.1,
    }
    raw_plan = yield ctx.call_activity(llm_activity, input=plan_prompt, retry_policy=LLM_RETRY)
    assignments = _parse_assignments(raw_plan, complex_task)

    ctx.signal_entity(entity_id, "set_status", {"agent": "coordinator", "status": "dispatching"})
    tasks = [
        ctx.call_sub_orchestrator(
            agent,
            input={
                "complex_task": complex_task,
                "role": assignment.role,
                "goal": assignment.goal,
                "tools": assignment.tools,
                "shared_entity_key": ctx.instance_id,
                "max_iterations": 2,
            },
            instance_id=f"{ctx.instance_id}:{assignment.role.lower()}",
            retry_policy=LLM_RETRY,
        )
        for assignment in assignments
    ]
    agent_results = yield task.when_all(tasks)

    shared_snapshot = yield ctx.call_entity(entity_id, "snapshot")
    final_prompt = {
        "system_prompt": "Produce a final synthesis from multi-agent outputs and shared state.",
        "user_prompt": (
            f"Complex task: {complex_task}\n\n"
            f"Agent results: {json.dumps(agent_results)}\n\n"
            f"Shared entity snapshot: {json.dumps(shared_snapshot)}\n\n"
            "Provide a concise combined result with recommendations."
        ),
        "temperature": 0.2,
    }
    final_result = yield ctx.call_activity(llm_activity, input=final_prompt, retry_policy=LLM_RETRY)
    ctx.signal_entity(entity_id, "set_status", {"agent": "coordinator", "status": "completed"})
    return {
        "task": complex_task,
        "assignments": [assignment.model_dump() for assignment in assignments],
        "shared_state": shared_snapshot,
        "agent_results": agent_results,
        "final_result": final_result,
    }
