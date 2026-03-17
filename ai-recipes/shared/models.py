"""Shared data models for Durable Task AI Hub recipes."""

from dataclasses import dataclass, field
from typing import Any


@dataclass
class LlmRequest:
    """Request payload for LLM invocation activities."""

    model: str = "gpt-5.4"
    instructions: str = "You are a helpful assistant."
    input: Any = ""
    tools: list[dict[str, Any]] = field(default_factory=list)
    response_format: str | None = None


@dataclass
class LlmResponse:
    """Response from an LLM invocation activity."""

    content: str = ""
    tool_calls: list[dict[str, Any]] = field(default_factory=list)
    raw_output: list[dict[str, Any]] = field(default_factory=list)


@dataclass
class ToolCall:
    """A single tool call from an LLM response."""

    name: str
    arguments: dict[str, Any]
    call_id: str = ""


@dataclass
class Message:
    """A message in a conversation history."""

    role: str  # "user", "assistant", "tool", "system"
    content: str
    tool_call_id: str | None = None
    name: str | None = None
