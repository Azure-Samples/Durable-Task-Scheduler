from __future__ import annotations

import json
import os
import sys
from typing import Optional

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations.agent import agent_orchestration

DEFAULT_QUERY = "What does 'ephemeral' mean?"
ALTERNATE_EXAMPLES = [
    'Convert 72 degrees Fahrenheit to Celsius',
    'Tell me something interesting',
]


def get_connection_config() -> dict[str, object]:
    endpoint = os.getenv('ENDPOINT', 'http://localhost:8080')
    taskhub = os.getenv('TASKHUB', 'default')
    return {
        'host_address': endpoint,
        'secure_channel': endpoint != 'http://localhost:8080',
        'taskhub': taskhub,
        'token_credential': None,
    }


def _format_output(serialized_output: Optional[str]) -> str:
    if serialized_output is None:
        return ''
    try:
        parsed = json.loads(serialized_output)
    except json.JSONDecodeError:
        return serialized_output
    if isinstance(parsed, str):
        return parsed
    return json.dumps(parsed, indent=2)


def main() -> None:
    if len(sys.argv) < 2:
        user_query = DEFAULT_QUERY
        print(f'No query supplied; using default example: {DEFAULT_QUERY}')
        print('Other examples:')
        for example in ALTERNATE_EXAMPLES:
            print(f'  python client.py "{example}"')
    else:
        user_query = ' '.join(sys.argv[1:])

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(agent_orchestration, input=user_query)
    print(f'Scheduled instance: {instance_id}')

    state = client.wait_for_orchestration_completion(instance_id, timeout=120)
    if state is None:
        print('Timed out waiting for the orchestration to complete.')
        raise SystemExit(2)

    status_name = getattr(state.runtime_status, 'name', str(state.runtime_status))
    print(f'Status: {status_name}')

    if status_name == 'COMPLETED':
        print(_format_output(state.serialized_output))
        return

    print(getattr(state, 'serialized_output', None) or 'The orchestration did not complete successfully.')
    raise SystemExit(3)


if __name__ == '__main__':
    main()
