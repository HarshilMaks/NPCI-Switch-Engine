from datetime import datetime
from decimal import Decimal

from pydantic import BaseModel, Field


class ErrorEnvelope(BaseModel):
    code: str
    message: str
    details: dict = Field(default_factory=dict)
    correlation_id: str


class PaymentCreateRequest(BaseModel):
    payer_vpa: str
    payee_vpa: str
    amount: Decimal
    currency: str = "INR"
    client_ref: str | None = None


class PaymentResponse(BaseModel):
    transaction_id: str
    status: str
    accepted_at: datetime


class PaymentStatusResponse(BaseModel):
    transaction_id: str
    status: str
    amount: Decimal
    currency: str
    events: list[dict]


class ReversalRequest(BaseModel):
    original_transaction_id: str
    reason: str = "MANUAL_REVERSAL"


class ReconciliationRunResponse(BaseModel):
    run_id: str
    run_key: str
    status: str
    summary: dict

