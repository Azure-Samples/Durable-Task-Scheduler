from __future__ import annotations

import asyncio
import inspect
import json
from typing import Any, Optional, Union

from pydantic import BaseModel

from tools import get_handler


class ToolInvocationInput(BaseModel):
    tool_name: str
    arguments: Optional[Union[dict[str, Any], str]] = None


def _coerce_arguments(arguments: Optional[Union[dict[str, Any], str]]) -> dict[str, Any]:
    if arguments is None:
        return {}
    if isinstance(arguments, dict):
        return arguments
    parsed = json.loads(arguments)
    if not isinstance(parsed, dict):
        raise ValueError('Tool arguments must deserialize to an object.')
    return parsed


def invoke_tool(ctx: Any, payload: dict[str, Any]) -> str:
    request = ToolInvocationInput.model_validate(payload)
    handler = get_handler(request.tool_name)
    if handler is None:
        raise ValueError(f'Unknown tool: {request.tool_name}')

    result = handler(**_coerce_arguments(request.arguments))
    if inspect.isawaitable(result):
        result = asyncio.run(result)

    if isinstance(result, str):
        return result
    return json.dumps(result, ensure_ascii=False)
