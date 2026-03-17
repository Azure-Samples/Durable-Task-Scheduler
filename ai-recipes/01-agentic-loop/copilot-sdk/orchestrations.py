from __future__ import annotations

from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy

from activities import run_agent

DEFAULT_INPUT = "What does 'ephemeral' mean, and convert 100 km to miles"
RETRY = RetryPolicy(first_retry_interval=timedelta(seconds=3), max_number_of_attempts=3, backoff_coefficient=2.0)


def agent_orchestration(ctx: task.OrchestrationContext, user_input: str | None) -> str:
    """The entire agentic loop runs inside one durable activity.

    Copilot SDK handles tool selection and invocation internally.
    """
    user_input = user_input or DEFAULT_INPUT
    result = yield ctx.call_activity(run_agent, input=user_input, retry_policy=RETRY)
    return result
