from __future__ import annotations

import json
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import quote
from urllib.request import Request, urlopen

TOOL_DEFINITION: dict[str, Any] = {
    'type': 'function',
    'name': 'lookup_word',
    'description': 'Look up the definition of an English word, including part of speech and an example sentence when available.',
    'parameters': {
        'type': 'object',
        'properties': {
            'word': {
                'type': 'string',
                'description': 'The English word or short phrase to define.',
            }
        },
        'required': ['word'],
        'additionalProperties': False,
    },
}


def lookup_word(word: str) -> str:
    search_term = word.strip()
    if not search_term:
        return 'Please provide a word to define.'

    request = Request(
        f'https://api.dictionaryapi.dev/api/v2/entries/en/{quote(search_term)}',
        headers={
            'Accept': 'application/json',
            'User-Agent': 'durable-task-ai-hub/0.1',
        },
    )

    try:
        with urlopen(request, timeout=20) as response:
            payload = json.load(response)
    except HTTPError as exc:
        if exc.code == 404:
            return f'No dictionary entry was found for "{search_term}".'
        return f'Dictionary lookup failed with status {exc.code} for "{search_term}".'
    except URLError as exc:
        return f'Unable to reach the dictionary service: {exc.reason}.'

    if not isinstance(payload, list) or not payload:
        return f'The dictionary service returned an unexpected response for "{search_term}".'

    entry = payload[0]
    entry_word = entry.get('word', search_term)
    phonetic = entry.get('phonetic')
    meanings = entry.get('meanings', [])
    if not meanings:
        return f'No dictionary meanings were returned for "{entry_word}".'

    selected_meaning = meanings[0]
    definitions = selected_meaning.get('definitions', [])
    if not definitions:
        return f'No definitions were returned for "{entry_word}".'

    selected_definition = definitions[0]
    part_of_speech = selected_meaning.get('partOfSpeech', 'unknown')
    definition_text = selected_definition.get('definition', 'No definition available.')
    example = selected_definition.get('example')

    lines = [f'{entry_word} ({part_of_speech})']
    if phonetic:
        lines[0] += f' {phonetic}'
    lines.append(f'Definition: {definition_text}')
    if example:
        lines.append(f'Example: {example}')

    return '\n'.join(lines)
