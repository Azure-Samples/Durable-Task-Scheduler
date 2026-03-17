from durabletask import task

from models import TransferRequest


def notify_approvers_activity(ctx: task.ActivityContext, transfer_request_data: dict) -> None:
    transfer_request = TransferRequest.model_validate(transfer_request_data)
    print()
    print('=== FINANCIAL TRANSFER APPROVAL REQUIRED ===')
    print(f'Request ID   : {transfer_request.request_id}')
    print(f'Sender       : {transfer_request.sender}')
    print(f'Recipient    : {transfer_request.recipient}')
    print(f'Amount       : {transfer_request.amount:.2f} {transfer_request.currency}')
    print(f'Country      : {transfer_request.country}')
    print(f'Description  : {transfer_request.description}')
    print(f'Risk Level   : {transfer_request.risk_level}')
    print(f'Analysis     : {transfer_request.analysis}')
    print('Use send_approval.py to approve or reject this wire transfer.')
    print('============================================')
    print()
