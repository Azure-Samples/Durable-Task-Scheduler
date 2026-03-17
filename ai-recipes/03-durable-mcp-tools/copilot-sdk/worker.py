from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities import fetch_github_api
from orchestrations import GetRecentActivity, GetRepoInfo

DEFAULT_ENDPOINT = 'http://localhost:8080'
DEFAULT_TASKHUB = 'default'


def get_connection_settings() -> dict:
    endpoint = os.getenv('DTS_ENDPOINT') or os.getenv('ENDPOINT', DEFAULT_ENDPOINT)
    taskhub = os.getenv('DTS_TASKHUB') or os.getenv('TASKHUB', DEFAULT_TASKHUB)
    return {
        'host_address': endpoint,
        'taskhub': taskhub,
        'token_credential': None,
        'secure_channel': endpoint.startswith('https://'),
    }


def main() -> None:
    with DurableTaskSchedulerWorker(**get_connection_settings()) as worker:
        worker.add_orchestrator(GetRepoInfo)
        worker.add_orchestrator(GetRecentActivity)
        worker.add_activity(fetch_github_api)
        worker.start()

        print('Copilot SDK GitHub inspector worker is running. Press Ctrl+C to stop.')
        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print('Stopping worker...')


if __name__ == '__main__':
    main()
