from __future__ import annotations

import os
from datetime import timedelta
from pathlib import Path
from typing import Any

from dotenv import load_dotenv
from durabletask import task
from durabletask.task import RetryPolicy

from activities.llm_activity import invoke_llm
from activities.tool_invoker import invoke_tool
from tools import get_tools

load_dotenv(Path(__file__).resolve().parents[3] / ".env")

LLM_RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(seconds=120),
)
TOOL_RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=2),
    max_number_of_attempts=2,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=10),
    retry_timeout=timedelta(seconds=30),
)
AGENT_INSTRUCTIONS = (
    'You are a helpful agent. You can look up dictionary definitions, convert between common units, '
    'and fetch a surprising random fact. Use a tool when it materially improves the answer, but respond '
    'directly when the user does not need a tool. Once you have enough information, provide a concise final answer.'
)


def _extract_message_text(output_items: list[dict[str, Any]]) -> str:
    fragments: list[str] = []
    for item in output_items:
        if item.get('type') != 'message':
            continue
        for content_item in item.get('content', []):
            text = content_item.get('text') or content_item.get('value')
            if isinstance(text, str) and text:
                fragments.append(text)
    return '\n'.join(fragments).strip()


def agent_orchestration(ctx: task.OrchestrationContext, user_input: str) -> str:
    conversation_history: list[dict[str, Any]] = [{'role': 'user', 'content': user_input}]

    while True:
        ctx.set_custom_status({'turns': len(conversation_history)})
        llm_result = yield ctx.call_activity(
            invoke_llm,
            input={
                'model': os.getenv("OPENAI_MODEL", "gpt-5.4"),
                'instructions': AGENT_INSTRUCTIONS,
                'input': conversation_history,
                'tools': get_tools(),
            },
            retry_policy=LLM_RETRY_POLICY,
        )

        output_items = llm_result.get('output', [])
        tool_calls = [item for item in output_items if item.get('type') == 'function_call']

        if not tool_calls:
            final_text = llm_result.get('output_text') or _extract_message_text(output_items)
            return final_text or 'The agent completed without returning text.'

        for tool_call in tool_calls:
            conversation_history.append(tool_call)
            tool_result = yield ctx.call_activity(
                invoke_tool,
                input={
                    'tool_name': tool_call['name'],
                    'arguments': tool_call.get('arguments', '{}'),
                },
                retry_policy=TOOL_RETRY_POLICY,
            )
            conversation_history.append(
                {
                    'type': 'function_call_output',
                    'call_id': tool_call['call_id'],
                    'output': tool_result,
                }
            )
