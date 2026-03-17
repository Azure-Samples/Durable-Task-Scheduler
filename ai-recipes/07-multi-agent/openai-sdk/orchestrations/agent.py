from __future__ import annotations

import json
from datetime import timedelta
from typing import Any, Union

from durabletask import task
from durabletask.entities import EntityInstanceId
from durabletask.task import RetryPolicy
from pydantic import BaseModel, Field

from activities.llm_activity import llm_activity
from activities.tool_invoker import tool_invoker


LLM_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


class AgentInput(BaseModel):
    complex_task: str
    role: str
    goal: str
    tools: list[str] = Field(default_factory=list)
    shared_entity_key: str
    max_iterations: int = 2


class AgentDecision(BaseModel):
    tool_name: str
    tool_input: str
    done: bool = False


def _parse_decision(raw: str, role: str, complex_task: str, iteration: int) -> AgentDecision:
    try:
        return AgentDecision.model_validate(json.loads(raw))
    except Exception:
        tool_name = "search" if iteration == 1 else "summarize"
        return AgentDecision(
            tool_name=tool_name,
            tool_input=f"{role} perspective on {complex_task}",
            done=iteration >= 2,
        )


def agent(ctx: task.OrchestrationContext, input_: Union[dict[str, Any], AgentInput]):
    request = input_ if isinstance(input_, AgentInput) else AgentInput.model_validate(input_)
    entity_id = EntityInstanceId("sharedstate", request.shared_entity_key)
    ctx.signal_entity(entity_id, "set_status", {"agent": request.role, "status": "running"})
    findings: list[str] = []

    for iteration in range(1, request.max_iterations + 1):
        ctx.set_custom_status({"role": request.role, "iteration": iteration, "goal": request.goal})
        planning_prompt = {
            "system_prompt": (
                "You are a specialist agent. Return JSON with tool_name, tool_input, and done. "
                "Only choose from the tools provided."
            ),
            "user_prompt": (
                f"Complex task: {request.complex_task}\n"
                f"Role: {request.role}\n"
                f"Goal: {request.goal}\n"
                f"Available tools: {', '.join(request.tools)}\n"
                f"Iteration: {iteration}\n"
                f"Findings so far:\n" + ("\n\n".join(findings) if findings else "None yet.")
            ),
            "temperature": 0.1,
        }
        raw_decision = yield ctx.call_activity(llm_activity, input=planning_prompt, retry_policy=LLM_RETRY)
        decision = _parse_decision(raw_decision, request.role, request.complex_task, iteration)
        tool_result = yield ctx.call_activity(
            tool_invoker,
            input={
                "tool_name": decision.tool_name,
                "tool_input": decision.tool_input,
                "role": request.role,
            },
            retry_policy=LLM_RETRY,
        )
        finding = f"{request.role} iteration {iteration}: {tool_result}"
        findings.append(finding)
        ctx.signal_entity(
            entity_id,
            "add_finding",
            {
                "agent": request.role,
                "iteration": iteration,
                "finding": finding,
            },
        )
        if decision.done:
            break

    result = {
        "role": request.role,
        "goal": request.goal,
        "findings": findings,
    }
    ctx.signal_entity(entity_id, "set_status", {"agent": request.role, "status": "completed"})
    return result
