from datetime import datetime
from decimal import Decimal
from uuid import uuid4

from sqlalchemy import (
    DECIMAL,
    JSON,
    DateTime,
    ForeignKey,
    Integer,
    Numeric,
    String,
    UniqueConstraint,
)
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base


def now_utc() -> datetime:
    return datetime.utcnow()


class Account(Base):
    __tablename__ = "accounts"

    id: Mapped[str] = mapped_column(String(64), primary_key=True)
    user_id: Mapped[str] = mapped_column(String(128), nullable=False)
    currency: Mapped[str] = mapped_column(String(8), nullable=False, default="INR")
    status: Mapped[str] = mapped_column(String(32), nullable=False, default="ACTIVE")
    available_balance: Mapped[Decimal] = mapped_column(
        DECIMAL(18, 2), nullable=False, default=Decimal("0.00")
    )
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)


class VPA(Base):
    __tablename__ = "vpas"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    handle: Mapped[str] = mapped_column(String(128), unique=True, nullable=False)
    account_id: Mapped[str] = mapped_column(
        String(64), ForeignKey("accounts.id"), nullable=False, index=True
    )
    status: Mapped[str] = mapped_column(String(32), nullable=False, default="ACTIVE")
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)

    account: Mapped[Account] = relationship(Account)


class Transaction(Base):
    __tablename__ = "transactions"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    payer_vpa: Mapped[str] = mapped_column(String(128), nullable=False)
    payee_vpa: Mapped[str] = mapped_column(String(128), nullable=False)
    amount: Mapped[Decimal] = mapped_column(DECIMAL(18, 2), nullable=False)
    currency: Mapped[str] = mapped_column(String(8), nullable=False)
    status: Mapped[str] = mapped_column(String(32), nullable=False, index=True)
    version: Mapped[int] = mapped_column(Integer, nullable=False, default=1)
    idempotency_key: Mapped[str] = mapped_column(String(128), nullable=False, unique=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=now_utc, onupdate=now_utc
    )


class TransactionEvent(Base):
    __tablename__ = "transaction_events"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    transaction_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("transactions.id"), nullable=False, index=True
    )
    from_status: Mapped[str] = mapped_column(String(32), nullable=False)
    to_status: Mapped[str] = mapped_column(String(32), nullable=False)
    reason_code: Mapped[str] = mapped_column(String(64), nullable=False)
    actor: Mapped[str] = mapped_column(String(64), nullable=False)
    metadata_json: Mapped[dict] = mapped_column(JSON, nullable=False, default=dict)
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)


class LedgerEntry(Base):
    __tablename__ = "ledger_entries"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    transaction_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("transactions.id"), nullable=False, index=True
    )
    account_id: Mapped[str] = mapped_column(
        String(64), ForeignKey("accounts.id"), nullable=False, index=True
    )
    leg_type: Mapped[str] = mapped_column(String(16), nullable=False)  # DEBIT / CREDIT
    amount: Mapped[Decimal] = mapped_column(DECIMAL(18, 2), nullable=False)
    currency: Mapped[str] = mapped_column(String(8), nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)


class Reversal(Base):
    __tablename__ = "reversals"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    original_transaction_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    reversal_transaction_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    reason: Mapped[str] = mapped_column(String(128), nullable=False)
    status: Mapped[str] = mapped_column(String(32), nullable=False, default="REVERSED")
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)


class IdempotencyRecord(Base):
    __tablename__ = "idempotency_records"
    __table_args__ = (
        UniqueConstraint("idempotency_key", "scope_key", name="uq_idempotency_scope"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    idempotency_key: Mapped[str] = mapped_column(String(128), nullable=False, index=True)
    scope_key: Mapped[str] = mapped_column(String(64), nullable=False, index=True)
    request_hash: Mapped[str] = mapped_column(String(128), nullable=False)
    response_payload: Mapped[dict] = mapped_column(JSON, nullable=False)
    status_code: Mapped[int] = mapped_column(Integer, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)


class OutboxEvent(Base):
    __tablename__ = "outbox_events"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    aggregate_type: Mapped[str] = mapped_column(String(64), nullable=False)
    aggregate_id: Mapped[str] = mapped_column(String(64), nullable=False, index=True)
    event_type: Mapped[str] = mapped_column(String(64), nullable=False, index=True)
    payload: Mapped[dict] = mapped_column(JSON, nullable=False, default=dict)
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)
    published_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)


class ReconciliationRun(Base):
    __tablename__ = "reconciliation_runs"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    run_key: Mapped[str] = mapped_column(String(64), nullable=False, unique=True)
    status: Mapped[str] = mapped_column(String(32), nullable=False, default="COMPLETED")
    summary_json: Mapped[dict] = mapped_column(JSON, nullable=False, default=dict)
    started_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)
    completed_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)


class ReconciliationDiff(Base):
    __tablename__ = "reconciliation_diffs"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=lambda: str(uuid4()))
    run_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("reconciliation_runs.id"), nullable=False, index=True
    )
    transaction_id: Mapped[str] = mapped_column(String(36), nullable=False, index=True)
    diff_type: Mapped[str] = mapped_column(String(64), nullable=False)
    details_json: Mapped[dict] = mapped_column(JSON, nullable=False, default=dict)
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=now_utc)

