from __future__ import annotations

from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy

from activities import send_agent_message


DEFAULT_MESSAGES = [
    "What is Durable Task?",
    "How does durable execution help after a worker restart?",
    "Give me a code example in Python.",
]

TURN_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=3),
    max_number_of_attempts=3,
)


def durable_conversation(
    ctx: task.OrchestrationContext, messages: list[str] | None = None
) -> list[str]:
    """Run a multi-turn Copilot conversation as durable activity turns."""
    prompts = messages or DEFAULT_MESSAGES
    session_id = f"durable-session-{ctx.instance_id}"
    responses: list[str] = []

    for index, message in enumerate(prompts, start=1):
        ctx.set_custom_status(
            {
                "stage": "conversation",
                "message_index": index,
                "message_count": len(prompts),
                "session_id": session_id,
            }
        )
        response = yield ctx.call_activity(
            send_agent_message,
            input={"session_id": session_id, "prompt": message},
            retry_policy=TURN_RETRY,
        )
        responses.append(response)

    return responses
