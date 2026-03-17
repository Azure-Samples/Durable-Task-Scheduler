from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities import analyze_request, execute_transfer, notify_approvers
from orchestrations import transfer_approval_orchestration

LOCAL_ENDPOINT = 'http://localhost:8080'


def get_connection_config() -> dict:
    endpoint = os.getenv('ENDPOINT', LOCAL_ENDPOINT)
    taskhub = os.getenv('TASKHUB', 'default')
    return {
        'host_address': endpoint,
        'secure_channel': endpoint != LOCAL_ENDPOINT,
        'taskhub': taskhub,
        'token_credential': None,
    }


def main() -> None:
    with DurableTaskSchedulerWorker(**get_connection_config()) as worker:
        worker.add_orchestrator(transfer_approval_orchestration)
        worker.add_activity(analyze_request)
        worker.add_activity(notify_approvers)
        worker.add_activity(execute_transfer)
        worker.start()

        print('Copilot SDK human-in-the-loop worker is running. Press Ctrl+C to stop.')
        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print('Stopping worker...')


if __name__ == '__main__':
    main()
