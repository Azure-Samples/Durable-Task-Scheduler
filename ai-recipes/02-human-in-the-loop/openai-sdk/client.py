import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations.approval_workflow import approval_workflow

DEFAULT_TRANSFER_REQUEST = (
    'Transfer $50,000 to account IBAN DE89370400440532013000 in Germany for consulting services'
)
LOW_RISK_EXAMPLE = 'Transfer $500 to account 12345678 for office supplies'


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
    if len(sys.argv) < 2:
        user_request = DEFAULT_TRANSFER_REQUEST
        print('No transfer request supplied; using default high-risk example:')
        print(f'  {DEFAULT_TRANSFER_REQUEST}')
        print('Low-risk example:')
        print(f'  python3 client.py "{LOW_RISK_EXAMPLE}"')
    else:
        user_request = ' '.join(sys.argv[1:]).strip()

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(approval_workflow, input=user_request)

    print(f'Scheduled transfer approval workflow: {instance_id}')
    print('If the transfer requires approval, send a decision with:')
    print(f'  python3 send_approval.py {instance_id} approve "Approved after finance review"')
    print(f'  python3 send_approval.py {instance_id} reject "Recipient verification failed"')


if __name__ == '__main__':
    main()
