from __future__ import annotations

import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations import transfer_approval_orchestration

DEFAULT_TRANSFER_REQUEST = (
    'Transfer $50,000 to account IBAN DE89370400440532013000 in Germany for consulting services'
)
LOW_RISK_EXAMPLE = 'Transfer $500 to account 12345678 for office supplies'
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
    if len(sys.argv) < 2:
        request = DEFAULT_TRANSFER_REQUEST
        print('No transfer request supplied; using default high-risk example:')
        print(f'  {DEFAULT_TRANSFER_REQUEST}')
        print('Low-risk example:')
        print(f'  python3 client.py "{LOW_RISK_EXAMPLE}"')
    else:
        request = ' '.join(sys.argv[1:]).strip()

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(transfer_approval_orchestration, input=request)
    print(f'Scheduled Copilot SDK transfer approval workflow: {instance_id}')
    print('If approval is required, send a decision with:')
    print(f'  python3 send_approval.py {instance_id} approve "Approved after finance review"')
    print(f'  python3 send_approval.py {instance_id} reject "Recipient verification failed"')


if __name__ == '__main__':
    main()
