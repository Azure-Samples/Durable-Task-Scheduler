from __future__ import annotations

from collections.abc import Callable
from typing import Any, Optional

from .dictionary import TOOL_DEFINITION as DICTIONARY_TOOL_DEFINITION
from .dictionary import lookup_word
from .random_fact import TOOL_DEFINITION as RANDOM_FACT_TOOL_DEFINITION
from .random_fact import get_random_fact
from .unit_converter import TOOL_DEFINITION as UNIT_CONVERTER_TOOL_DEFINITION
from .unit_converter import convert_units

_TOOLS: dict[str, tuple[dict[str, Any], Callable[..., Any]]] = {
    'lookup_word': (DICTIONARY_TOOL_DEFINITION, lookup_word),
    'convert_units': (UNIT_CONVERTER_TOOL_DEFINITION, convert_units),
    'get_random_fact': (RANDOM_FACT_TOOL_DEFINITION, get_random_fact),
}


def get_tools() -> list[dict[str, Any]]:
    return [definition for definition, _ in _TOOLS.values()]


def get_handler(tool_name: str) -> Optional[Callable[..., Any]]:
    tool_entry = _TOOLS.get(tool_name)
    if tool_entry is None:
        return None
    return tool_entry[1]
