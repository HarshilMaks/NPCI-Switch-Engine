# Design Doc 01 — System Design Overview

## 1. Objective

Provide implementation-level design for a UPI-inspired payment rail simulator that guarantees:

1. financial correctness
2. deterministic execution under retries/failures
3. complete auditability

## 2. Core modules

1. **Payment API**: accepts payment intents with idempotency.
2. **State Orchestrator**: drives transaction lifecycle and compensation.
3. **Ledger Engine**: append-only double-entry posting.
4. **Outbox Relay**: DB-to-event-bus consistency bridge.
5. **Reconciliation Engine**: validates end-of-flow consistency.
6. **Risk Engine**: policy checks (velocity/amount/holds).
7. **Admin Ops API**: operational visibility and controls.

## 3. Execution model

1. API writes transaction + outbox in one DB transaction.
2. Outbox relay publishes domain event.
3. Orchestrator consumes event and performs payer-debit/payee-credit flow.
4. On failure after debit, orchestrator triggers reversal aggregate.
5. Ledger records all monetary movements immutably.
6. Reconciliation compares transactional records and reports diffs.

## 4. State model

`INITIATED -> AUTH_PENDING -> AUTHORIZED -> DEBIT_POSTED -> CREDIT_POSTED -> COMPLETED`  
Failure/compensation paths:  
`... -> FAILED` and `DEBIT_POSTED -> REVERSAL_PENDING -> REVERSED | REVERSAL_FAILED`

## 5. Design constraints

1. No ledger mutation after insert.
2. No illegal state transitions.
3. No duplicate financial effects.
4. All transitions/events must be traceable by `transaction_id` and `correlation_id`.

## 6. Implementation order

1. Payment aggregate + API contracts
2. Idempotency layer
3. State machine guard
4. Ledger posting and invariants
5. Outbox + worker orchestration
6. Reversal flows
7. Reconciliation + ops endpoints

