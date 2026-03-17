from pydantic import BaseModel, Field


class TransferRequest(BaseModel):
    request_id: str = Field(description='Unique ID for this transfer review request')
    sender: str = Field(description='Account or business entity sending the funds')
    recipient: str = Field(description='Recipient account, vendor, or beneficiary receiving the funds')
    amount: float = Field(description='Transfer amount in the stated currency')
    currency: str = Field(description='ISO currency code for the transfer')
    country: str = Field(description='Destination country for the transfer')
    description: str = Field(description='Business purpose or memo for the transfer')
    risk_level: str = Field(description='Risk level assigned by the transfer analysis step')
    analysis: str = Field(description='Why the transfer was classified with that risk level')


class ApprovalDecision(BaseModel):
    request_id: str = Field(description='Transfer request being approved or rejected')
    approved: bool = Field(description='Whether the wire transfer is approved')
    reason: str = Field(description='Why the reviewer approved or rejected the transfer')
    approver: str = Field(description='Human or system identity that made the transfer decision')
