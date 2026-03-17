from __future__ import annotations

import json
import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

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
    if len(sys.argv) < 4:
        print('Usage: python3 send_approval.py <instance_id> <approve|reject> <reason>')
        raise SystemExit(1)

    instance_id = sys.argv[1]
    decision_word = sys.argv[2].strip().lower()
    reason = ' '.join(sys.argv[3:]).strip()

    if decision_word not in {'approve', 'reject'}:
        print("Decision must be either 'approve' or 'reject'.")
        raise SystemExit(1)

    client = DurableTaskSchedulerClient(**get_connection_config())
    state = client.get_orchestration_state(instance_id)
    if state and state.serialized_custom_status:
        try:
            print(f'Current status: {json.loads(state.serialized_custom_status)}')
        except json.JSONDecodeError:
            pass

    decision = {
        'approved': decision_word == 'approve',
        'reason': reason,
        'approver': os.getenv('APPROVER', 'finance-reviewer'),
    }
    client.raise_orchestration_event(instance_id, 'approval', data=decision)
    print(f"Sent {decision_word} decision to orchestration {instance_id}.")


if __name__ == '__main__':
    main()
