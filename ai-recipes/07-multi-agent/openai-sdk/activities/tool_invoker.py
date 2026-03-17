from __future__ import annotations

from typing import Any, Union

from durabletask import task
from pydantic import BaseModel, Field


class ToolInvocation(BaseModel):
    tool_name: str
    tool_input: str
    role: str = Field(default="Agent")


def tool_invoker(ctx: task.ActivityContext, input_: Union[dict[str, Any], ToolInvocation]) -> str:
    request = input_ if isinstance(input_, ToolInvocation) else ToolInvocation.model_validate(input_)
    tool_name = request.tool_name.lower()
    if tool_name == "search":
        return (
            f"[{request.role}] Search results for \"{request.tool_input}\": "
            "three relevant examples, two implementation notes, and one caution about operational complexity."
        )
    if tool_name == "analyze":
        return (
            f"[{request.role}] Analysis of \"{request.tool_input}\": "
            "the strongest option improves reliability but increases coordination overhead."
        )
    if tool_name == "plan":
        return (
            f"[{request.role}] Plan for \"{request.tool_input}\": "
            "define milestones, assign responsibilities, and checkpoint outcomes in shared durable state."
        )
    return f"[{request.role}] Summary for \"{request.tool_input}\": consolidate findings into one concise recommendation."
