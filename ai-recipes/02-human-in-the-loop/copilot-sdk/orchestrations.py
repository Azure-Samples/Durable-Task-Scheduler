from __future__ import annotations

from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy

from activities import analyze_request, execute_transfer, notify_approvers

RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=2),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


def _wait_for_approval(ctx: task.OrchestrationContext):
    approval_event = ctx.wait_for_external_event('approval')
    timeout_task = ctx.create_timer(ctx.current_utc_datetime + timedelta(minutes=5))
    winner = yield task.when_any([approval_event, timeout_task])
    if winner == timeout_task:
        raise TimeoutError('Timed out waiting for approval.')
    return approval_event.get_result()


def transfer_approval_orchestration(ctx: task.OrchestrationContext, request: str) -> str:
    analysis = yield ctx.call_activity(analyze_request, input=request, retry_policy=RETRY)
    ctx.set_custom_status(
        {
            'status': 'analyzed',
            'request_id': analysis.get('request_id'),
            'risk_level': analysis.get('risk_level'),
            'proposed_action': analysis.get('proposed_action'),
        }
    )

    if analysis.get('risk_level') == 'HIGH_RISK':
        yield ctx.call_activity(notify_approvers, input=analysis, retry_policy=RETRY)
        ctx.set_custom_status(
            {
                'status': 'awaiting_approval',
                'request_id': analysis.get('request_id'),
                'risk_level': analysis.get('risk_level'),
            }
        )
        try:
            decision = yield from _wait_for_approval(ctx)
        except TimeoutError:
            ctx.set_custom_status({'status': 'timed_out', 'request_id': analysis.get('request_id')})
            return 'Transfer timed out waiting for approval'

        decision = decision or {}
        if not decision.get('approved'):
            ctx.set_custom_status(
                {
                    'status': 'rejected',
                    'request_id': analysis.get('request_id'),
                    'reason': decision.get('reason', 'No reason supplied'),
                }
            )
            return f"Transfer rejected: {decision.get('reason', 'No reason supplied')}"

        analysis = {**analysis, 'approval': decision}

    result = yield ctx.call_activity(execute_transfer, input=analysis, retry_policy=RETRY)
    ctx.set_custom_status({'status': 'completed', 'request_id': analysis.get('request_id')})
    return result
