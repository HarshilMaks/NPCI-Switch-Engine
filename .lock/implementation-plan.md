# Implementation Plan (Execution Root)

**Last updated**: 2026-04-29  
**Current phase**: Phase 1-2 (Foundation + Core APIs) — IN PROGRESS

---

## Phase 1 — Foundation ✓ COMPLETE

- [x] Project skeleton (cmd/server, internal packages)
- [x] Config loading and environment setup
- [x] SQLite database migrations (10 tables: accounts, vpas, transactions, events, ledger, reversals, idempotency, outbox, recon_runs, recon_diffs)
- [x] Health/readiness endpoints
- [x] Dependency wiring (chi router, services, handlers)

---

## Phase 2 — Core domain and APIs ✓ COMPLETE

- [x] Transaction aggregate + state machine (INITIATED → AUTH_PENDING → AUTHORIZED → DEBIT_POSTED → CREDIT_POSTED → COMPLETED)
- [x] Idempotent `POST /api/v1/payments` with Idempotency-Key header
- [x] `GET /api/v1/payments/{id}` with full event history
- [x] State transition guard module
- [x] AppError wrapper for consistent error responses
- [x] Seed data (alice@bank, bob@bank with 1M balance each)

---

## Phase 3 — Ledger core ✓ COMPLETE

- [x] Append-only `ledger_entries` table
- [x] Double-entry posting service (insertLedgerPair)
- [x] Balance update logic (applyBalanceUpdates)
- [x] Invariant: debit total == credit total for all settled transactions
- [x] Balance enforcement (insufficient funds check)

---

## Phase 4 — Async architecture ⧗ IN PROGRESS

- [x] Outbox table (`outbox_events`) for event propagation
- [x] Insert outbox events on every state transition
- [ ] Outbox relay worker (goroutine pulling and publishing)
- [ ] Kafka/NATS integration (deferred to phase 5)
- [ ] Event schema and consumer scaffold

---

## Phase 5 — Failure handling IN PROGRESS

- [x] Error classification (PAYER_VPA_NOT_FOUND, INSUFFICIENT_FUNDS, PAYEE_UNAVAILABLE, etc.)
- [x] Manual reversal endpoint `POST /api/v1/reversals`
- [x] Auto-reversal on credit posting failure
- [ ] Retry policy and exponential backoff
- [ ] DLQ handling for poison events (phase 2 worker)

---

## Phase 6 — Reconciliation and ops ✓ COMPLETE

- [x] Reconciliation runner (`POST /api/v1/reconciliation/run`)
- [x] Mismatch types: missing_leg, amount_mismatch, stale_pending, unpublished_outbox_events
- [ ] Admin endpoints for transaction timelines and incident review
- [ ] Operational dashboards and alert wiring

---

## Phase 7 — Hardening IN PROGRESS

- [x] Security controls baseline (parameterized queries, decimal precision)
- [ ] Unit tests for critical paths
- [ ] Load testing under duplicate storms
- [ ] Documentation consistency pass (locked docs updated)

---

## Next steps (immediate)

1. **Write unit tests** for payment service methods (idempotency, state transitions, ledger invariants)
2. **Implement outbox relay worker** (goroutine polling and publishing outbox_events)
3. **Add benchmark suite** for API latency and reconciliation performance
4. **Create smoke test CLI** (standalone tool for rapid validation)
5. **Update API spec** (.lock/api-spec.md) with actual response examples

---

## Definition of done per phase

1. Code complete and compiles
2. Critical tests pass locally
3. `.lock/` docs updated to reflect changes
4. No unresolved invariant regressions
5. Server builds single binary, runs with zero external dependencies (MVP)

