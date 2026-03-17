from __future__ import annotations

import json
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

TOOL_DEFINITION: dict[str, Any] = {
    'type': 'function',
    'name': 'get_random_fact',
    'description': 'Fetch a surprising random fact to add variety or inspiration to a conversation.',
    'parameters': {
        'type': 'object',
        'properties': {},
        'required': [],
        'additionalProperties': False,
    },
}


def get_random_fact() -> str:
    request = Request(
        'https://uselessfacts.jsph.pl/api/v2/facts/random',
        headers={
            'Accept': 'application/json',
            'User-Agent': 'durable-task-ai-hub/0.1',
        },
    )

    try:
        with urlopen(request, timeout=20) as response:
            payload = json.load(response)
    except HTTPError as exc:
        return f'Random fact request failed with status {exc.code}.'
    except URLError as exc:
        return f'Unable to reach the random fact service: {exc.reason}.'

    fact_text = payload.get('text')
    if not fact_text:
        return 'The random fact service returned an empty response.'

    source_url = payload.get('source_url') or payload.get('permalink')
    if source_url:
        return f'{fact_text}\nSource: {source_url}'
    return fact_text
