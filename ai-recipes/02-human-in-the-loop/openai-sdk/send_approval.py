import json
import os
import sys

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from models import ApprovalDecision

EXAMPLE_APPROVE = 'python3 send_approval.py <instance_id> approve "Approved after finance review"'
EXAMPLE_REJECT = 'python3 send_approval.py <instance_id> reject "Recipient verification failed"'


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
    if len(sys.argv) < 4:
        print('Usage: python3 send_approval.py <instance_id> <approve|reject> <reason>')
        print('Approve example:')
        print(f'  {EXAMPLE_APPROVE}')
        print('Reject example:')
        print(f'  {EXAMPLE_REJECT}')
        raise SystemExit(1)

    instance_id = sys.argv[1]
    decision_word = sys.argv[2].lower()
    reason = ' '.join(sys.argv[3:]).strip()

    if decision_word not in {'approve', 'reject'}:
        print("Decision must be either 'approve' or 'reject'.")
        raise SystemExit(1)

    client = DurableTaskSchedulerClient(**get_connection_config())
    state = client.get_orchestration_state(instance_id)
    request_id = instance_id

    if state and state.serialized_custom_status:
        try:
            custom_status = json.loads(state.serialized_custom_status)
            request_id = custom_status.get('request_id', request_id)
            print(f"Current transfer status: {custom_status.get('status', 'unknown')}")
        except json.JSONDecodeError:
            pass

    decision = ApprovalDecision(
        request_id=request_id,
        approved=decision_word == 'approve',
        reason=reason,
        approver=os.getenv('APPROVER', 'finance-reviewer'),
    )

    client.raise_orchestration_event(
        instance_id,
        'approval_decision',
        data=decision.model_dump(),
    )
    print(f"Sent {decision_word} decision for transfer workflow {instance_id}.")


if __name__ == '__main__':
    main()
