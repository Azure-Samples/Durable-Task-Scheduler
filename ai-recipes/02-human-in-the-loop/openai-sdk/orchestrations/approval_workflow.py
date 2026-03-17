from datetime import timedelta
from typing import Any

from durabletask import task

from activities.execute_action import process_wire_transfer_activity
from activities.llm_activity import analyze_transfer_request_activity
from activities.notify_approvers import notify_approvers_activity
from models import ApprovalDecision, TransferRequest

HIGH_VALUE_THRESHOLD = 10_000.0
DOMESTIC_COUNTRIES = {'us', 'usa', 'united states', 'united states of america'}


def _is_international_transfer(transfer_request: TransferRequest) -> bool:
    return transfer_request.country.strip().lower() not in DOMESTIC_COUNTRIES


def _requires_human_approval(transfer_request: TransferRequest) -> bool:
    return transfer_request.amount > HIGH_VALUE_THRESHOLD or _is_international_transfer(transfer_request)


def approval_workflow(ctx: task.OrchestrationContext, user_request: str) -> dict[str, Any]:
    analysis_payload = yield ctx.call_activity(analyze_transfer_request_activity, input=user_request)
    transfer_request = TransferRequest.model_validate(analysis_payload)
    approval_required = _requires_human_approval(transfer_request)

    if approval_required:
        ctx.set_custom_status(
            {
                'status': 'awaiting_approval',
                'request_id': transfer_request.request_id,
                'recipient': transfer_request.recipient,
                'amount': transfer_request.amount,
                'currency': transfer_request.currency,
                'country': transfer_request.country,
            }
        )
        yield ctx.call_activity(notify_approvers_activity, input=transfer_request.model_dump())

        approval_event = ctx.wait_for_external_event('approval_decision')
        timeout_task = ctx.create_timer(ctx.current_utc_datetime + timedelta(minutes=5))
        winner = yield task.when_any([approval_event, timeout_task])

        if winner == timeout_task:
            ctx.set_custom_status(
                {
                    'status': 'timed_out',
                    'request_id': transfer_request.request_id,
                    'recipient': transfer_request.recipient,
                }
            )
            return {
                'status': 'timed_out',
                'request_id': transfer_request.request_id,
                'transfer_request': transfer_request.model_dump(),
                'message': 'Timed out waiting for wire transfer approval after 5 minutes.',
            }

        decision_payload = approval_event.get_result()
        approval_decision = ApprovalDecision.model_validate(decision_payload)

        if approval_decision.request_id != transfer_request.request_id:
            raise ValueError(
                'Approval decision request_id does not match the pending transfer request.'
            )

        if not approval_decision.approved:
            ctx.set_custom_status(
                {
                    'status': 'rejected',
                    'request_id': transfer_request.request_id,
                    'approver': approval_decision.approver,
                }
            )
            return {
                'status': 'rejected',
                'approval_required': True,
                'transfer_request': transfer_request.model_dump(),
                'approval_decision': approval_decision.model_dump(),
            }

        execution_result = yield ctx.call_activity(
            process_wire_transfer_activity,
            input={
                'transfer_request': transfer_request.model_dump(),
                'approval_decision': approval_decision.model_dump(),
            },
        )
        ctx.set_custom_status(
            {
                'status': 'completed',
                'request_id': transfer_request.request_id,
                'approver': approval_decision.approver,
            }
        )
        return {
            'status': 'approved',
            'approval_required': True,
            'transfer_request': transfer_request.model_dump(),
            'approval_decision': approval_decision.model_dump(),
            'execution_result': execution_result,
        }

    system_decision = ApprovalDecision(
        request_id=transfer_request.request_id,
        approved=True,
        reason='Auto-approved because the transfer is domestic and below the $10,000 threshold.',
        approver='system',
    )
    execution_result = yield ctx.call_activity(
        process_wire_transfer_activity,
        input={
            'transfer_request': transfer_request.model_dump(),
            'approval_decision': system_decision.model_dump(),
        },
    )
    ctx.set_custom_status(
        {
            'status': 'completed',
            'request_id': transfer_request.request_id,
            'approver': 'system',
        }
    )
    return {
        'status': 'approved',
        'approval_required': False,
        'transfer_request': transfer_request.model_dump(),
        'approval_decision': system_decision.model_dump(),
        'execution_result': execution_result,
    }
