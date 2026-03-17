from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.llm_activity import invoke_llm
from activities.tool_invoker import invoke_tool
from orchestrations.agent import agent_orchestration


def get_connection_config() -> dict[str, object]:
    endpoint = os.getenv('ENDPOINT', 'http://localhost:8080')
    taskhub = os.getenv('TASKHUB', 'default')
    return {
        'host_address': endpoint,
        'secure_channel': endpoint != 'http://localhost:8080',
        'taskhub': taskhub,
        'token_credential': None,
    }


def main() -> None:
    config = get_connection_config()
    with DurableTaskSchedulerWorker(**config) as worker:
        worker.add_orchestrator(agent_orchestration)
        worker.add_activity(invoke_llm)
        worker.add_activity(invoke_tool)
        worker.start()

        print('Recipe 01 worker is running.')
        print(f"Endpoint: {config['host_address']}")
        print(f"Task hub: {config['taskhub']}")
        print('Press Ctrl+C to stop.')

        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print('\nStopping worker...')


if __name__ == '__main__':
    main()
