# ADR Register

This file tracks key architecture decisions.

## ADR-001 — PostgreSQL as source of truth
- **Status**: Accepted
- **Decision**: Use PostgreSQL for transactions, events, ledger, reconciliation artifacts.
- **Why**: ACID guarantees and strong transactional semantics for money movement.

## ADR-002 — Append-only double-entry ledger
- **Status**: Accepted
- **Decision**: Ledger rows are immutable after insert.
- **Why**: Audit integrity and deterministic forensic replay.

## ADR-003 — Transactional outbox for event consistency
- **Status**: Accepted
- **Decision**: Domain state + outbox event written in same DB transaction.
- **Why**: Prevent DB/event divergence.

## ADR-004 — Saga orchestration with compensation
- **Status**: Accepted
- **Decision**: Multi-step payment flow with reversal compensation.
- **Why**: Safe distributed recovery for partial failures.

## ADR-005 — Idempotency mandatory on mutating APIs
- **Status**: Accepted
- **Decision**: Require `Idempotency-Key` for write operations.
- **Why**: Safe retries and duplicate suppression.

## ADR-006 — Redis for fast idempotency and throttling
- **Status**: Accepted
- **Decision**: Use Redis for key reservation and rate counters, backed by DB truth.
- **Why**: Low latency + survivable eventual persistence.

## ADR-007 — Kafka event bus for async workers
- **Status**: Accepted
- **Decision**: Use event bus decoupling for orchestration/reconciliation/notifications.
- **Why**: Scalability and resilient asynchronous processing.

