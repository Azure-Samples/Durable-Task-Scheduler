from __future__ import annotations

import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations import rag_orchestration

DEFAULT_QUERY = 'How does a durable RAG pipeline combine retrieval and generation?'
LOCAL_ENDPOINT = 'http://localhost:8080'


def get_connection_config() -> dict:
    endpoint = os.getenv('ENDPOINT', LOCAL_ENDPOINT)
    taskhub = os.getenv('TASKHUB', 'default')
    return {
        'host_address': endpoint,
        'taskhub': taskhub,
        'secure_channel': endpoint != LOCAL_ENDPOINT,
        'token_credential': None,
    }


def main() -> None:
    query = ' '.join(sys.argv[1:]).strip() or DEFAULT_QUERY
    if len(sys.argv) < 2:
        print('No query supplied; using default prompt:')
        print(f'  {DEFAULT_QUERY}')

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(rag_orchestration, input=query)
    print(f'Started Copilot SDK RAG orchestration with instance ID: {instance_id}')

    state = client.wait_for_orchestration_completion(instance_id, timeout=60)
    if state is None:
        raise TimeoutError('Timed out waiting for the Copilot SDK RAG orchestration to complete.')

    print('\nAnswer:\n')
    print(state.serialized_output)


if __name__ == '__main__':
    main()
