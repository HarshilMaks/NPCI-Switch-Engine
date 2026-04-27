# Implementation Plan (Execution Root)

## Phase 1 — Foundation
1. Project skeleton, config, env loading, dependency wiring.
2. DB migrations scaffold.
3. Health/readiness endpoints.

## Phase 2 — Core domain and APIs
1. Transaction aggregate + status model.
2. Idempotent `POST /payments` and `GET /payments/{id}`.
3. Transition guard module.

## Phase 3 — Ledger core
1. Append-only `ledger_entries`.
2. Posting service (debit/credit pair).
3. Invariant checks and failure blocking.

## Phase 4 — Async architecture
1. Outbox table + relay worker.
2. Event topics and consumer scaffold.
3. Orchestrator worker for debit/credit flow.

## Phase 5 — Failure handling
1. Retry policy and error classification.
2. Reversal aggregate + execution.
3. DLQ handling for poison events.

## Phase 6 — Reconciliation and ops
1. Reconciliation runner and mismatch types.
2. Admin endpoints for incidents/timelines.
3. Operational dashboards and alert wiring.

## Phase 7 — Hardening
1. Security controls baseline.
2. Load/concurrency tests.
3. Documentation consistency pass.

## Definition of done per phase
1. Code complete.
2. Tests for critical paths pass.
3. Docs updated in `.lock/`.
4. No unresolved invariant regressions.

