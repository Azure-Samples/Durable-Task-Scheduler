from datetime import datetime, timezone
from typing import Any

from durabletask import task

from models import ApprovalDecision, TransferRequest


def process_wire_transfer_activity(ctx: task.ActivityContext, execution_input: dict[str, Any]) -> dict[str, Any]:
    transfer_request = TransferRequest.model_validate(execution_input['transfer_request'])
    approval_decision = ApprovalDecision.model_validate(execution_input['approval_decision'])

    print(
        '[TRANSFER] '
        f"request_id={transfer_request.request_id} "
        f"sender={transfer_request.sender!r} "
        f"recipient={transfer_request.recipient!r} "
        f"amount={transfer_request.amount:.2f} {transfer_request.currency} "
        f"country={transfer_request.country!r} "
        f"approved_by={approval_decision.approver!r} "
        f"reason={approval_decision.reason!r}"
    )

    return {
        'status': 'processed',
        'request_id': transfer_request.request_id,
        'sender': transfer_request.sender,
        'recipient': transfer_request.recipient,
        'amount': transfer_request.amount,
        'currency': transfer_request.currency,
        'country': transfer_request.country,
        'description': transfer_request.description,
        'risk_level': transfer_request.risk_level,
        'approved_by': approval_decision.approver,
        'approval_reason': approval_decision.reason,
        'processed_at': datetime.now(timezone.utc).isoformat(),
    }
