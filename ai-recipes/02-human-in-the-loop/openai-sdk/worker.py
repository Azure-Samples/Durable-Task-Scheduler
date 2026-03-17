import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.execute_action import process_wire_transfer_activity
from activities.llm_activity import analyze_transfer_request_activity
from activities.notify_approvers import notify_approvers_activity
from orchestrations.approval_workflow import approval_workflow


def get_connection_config() -> dict:
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
    print(f"Connecting worker to {config['host_address']} (taskhub={config['taskhub']})")

    with DurableTaskSchedulerWorker(**config) as worker:
        worker.add_orchestrator(approval_workflow)
        worker.add_activity(analyze_transfer_request_activity)
        worker.add_activity(notify_approvers_activity)
        worker.add_activity(process_wire_transfer_activity)
        worker.start()

        print('Human-in-the-loop transfer worker is running. Press Ctrl+C to stop.')
        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print('Stopping worker...')


if __name__ == '__main__':
    main()
