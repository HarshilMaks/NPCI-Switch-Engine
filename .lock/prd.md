# PRD — UPI-Inspired Payment Rail Simulator (Backend-First)

## 1. Document control

- **Product**: UPI-Inspired Payment Rail Simulator
- **Type**: Product Requirements Document (PRD)
- **Version**: 1.0
- **Status**: Approved for implementation
- **Owner**: Core Backend Team
- **Linked docs**:
  - `.lock/architecture.md`
  - `.lock/npci-upi-system-architecture-and-simulator-blueprint.md`

---

## 2. Product summary

Build a production-grade simulator of an instant payment rail focused on financial correctness, reliability, and auditability.  
The system will support end-to-end payment lifecycle execution with:

1. immutable double-entry ledger
2. deterministic transaction state machine
3. idempotent request processing
4. reversal/refund compensation flows
5. reconciliation and operational observability

This is a **simulation platform**, not a live payment network integration.

---

## 3. Problem statement

Most backend portfolios show API CRUD, not payment-grade distributed consistency.  
This project solves that by simulating the hardest backend properties:

- money integrity under failures/retries
- exactly-once effect with at-least-once delivery
- full forensic audit trail
- realistic operational workflows (recon, incidents, reversals)

---

## 4. Vision

Create the best backend reference implementation for “how money moves safely in distributed systems,” with engineering quality high enough to be discussed in architecture interviews and backend system design rounds.

---

## 5. Goals and non-goals

## 5.1 Goals

1. Zero ledger integrity violations.
2. Deterministic behavior for retries/duplicates/timeouts.
3. Full transaction traceability from API call to terminal state.
4. Reconciliation workflow with actionable mismatch records.
5. Production-style ops signals (metrics, traces, alerts).

## 5.2 Non-goals

1. Integration with real NPCI/bank/card rails.
2. Processing real customer funds.
3. Regulatory/compliance certification claims.

---

## 6. Personas and user stories

## 6.1 Personas

1. **Backend engineer (primary)**: wants to build/test payment-grade flows.
2. **Interviewer/reviewer**: wants proof of distributed-systems depth.
3. **Operator/admin**: wants to inspect failures, retries, and reconciliation.

## 6.2 User stories

1. As an engineer, I can create a payment request and safely retry it without duplicate execution.
2. As an engineer, I can observe full transaction state transitions and linked ledger entries.
3. As an operator, I can detect and inspect reconciliation mismatches.
4. As an operator, I can trigger/manual-review reversals with complete audit trail.

---

## 7. Scope (MVP vs future)

## 7.1 MVP scope

1. Payment initiation + status APIs.
2. Transaction state machine orchestration worker.
3. Append-only double-entry ledger posting.
4. Idempotency key enforcement.
5. Automatic reversal after debit-success/credit-failure.
6. Reconciliation run + mismatch reporting.
7. Core admin endpoints for transaction and incident views.

## 7.2 Post-MVP scope

1. Offline/Bluetooth deferred-finality mode.
2. Risk scoring enhancements and policy tuning UI.
3. Advanced dispute workflow simulation.
4. Multi-tenant simulation profiles.

---

## 8. Functional requirements

## 8.1 Payment initiation

1. System shall expose `POST /api/v1/payments`.
2. Mutation requests shall require `Idempotency-Key`.
3. System shall return canonical prior response for duplicate idempotency key in same scope.
4. System shall create transaction aggregate in `INITIATED` and emit initiation event via outbox.

## 8.2 Payment orchestration

1. Worker shall transition transactions only through allowed state graph.
2. Worker shall invoke payer-debit then payee-credit legs through bank simulator adapters.
3. Worker shall apply retry policy only for retriable errors.
4. Worker shall finalize terminal states as immutable.

## 8.3 Ledger

1. System shall create balanced debit/credit entries for each monetary movement.
2. Ledger entries shall be append-only (no updates/deletes).
3. Sum(debits) must equal sum(credits) for completed/reversed transactions.

## 8.4 Reversal

1. System shall create compensating reversal transaction when debit succeeded but credit cannot complete.
2. Reversal transaction shall include reference to original transaction id.
3. Reversal lifecycle shall be auditable as independent aggregate.

## 8.5 Reconciliation

1. System shall run scheduled and on-demand reconciliation jobs.
2. System shall compare ledger outcomes vs settlement-like records.
3. System shall classify mismatches (`missing_leg`, `amount_mismatch`, `duplicate_posting`, `stale_pending`).
4. System shall store reconciliation runs and diff artifacts.

## 8.6 Observability

1. System shall emit structured logs with `transaction_id`, `correlation_id`.
2. System shall publish metrics for API latency, completion latency, reversal rate, recon mismatches.
3. System shall support tracing across API -> outbox -> worker -> adapter path.

---

## 9. Non-functional requirements

## 9.1 Correctness and reliability

1. No data corruption under worker restarts and message re-delivery.
2. No duplicate financial effects for repeated client requests.
3. Deterministic reconciliation outcomes for same input set.

## 9.2 Performance targets (simulator)

1. Payment create API p95 < 150 ms (normal load).
2. Happy-path async completion < 2 s (target).
3. Reconciliation job completion within configured operational window.

## 9.3 Security targets

1. TLS for all network traffic.
2. Secrets externalized from source repository.
3. Sensitive fields masked/tokenized in logs.
4. Role-based admin endpoints with audit records.

---

## 10. Data model requirements

Minimum entities:

1. `transactions`
2. `transaction_events`
3. `ledger_entries`
4. `idempotency_records`
5. `reversals`
6. `outbox_events`
7. `reconciliation_runs`
8. `reconciliation_diffs`
9. `accounts`
10. `vpas`

Mandatory invariants:

1. Immutable ledger row policy.
2. Single canonical response per idempotency key scope.
3. Strict allowed state transitions only.
4. Monotonic versioning for aggregate mutation.

---

## 11. API requirements

## 11.1 Core endpoints

1. `POST /api/v1/payments`
2. `GET /api/v1/payments/{transaction_id}`
3. `POST /api/v1/payments/{transaction_id}/confirm`
4. `POST /api/v1/payments/{transaction_id}/cancel`
5. `POST /api/v1/reversals`
6. `POST /api/v1/reconciliation/run`
7. `GET /api/v1/accounts/{account_id}/ledger`

## 11.2 API behavior

1. Standard error envelope with machine-readable error code.
2. Correlation ID returned in response headers.
3. Pagination support on ledger/events endpoints.

---

## 12. Eventing requirements

Required event topics:

1. `payment.initiated`
2. `payment.authorized`
3. `payment.debit_posted`
4. `payment.credit_posted`
5. `payment.completed`
6. `payment.failed`
7. `payment.reversal_pending`
8. `payment.reversed`
9. `reconciliation.diff_detected`

Event delivery requirements:

1. Transactional outbox pattern for publish consistency.
2. Consumer idempotency and DLQ support.

---

## 13. Risk and control requirements

1. Velocity rules at account/VPA level.
2. amount threshold rules with hold/reject actions.
3. Configurable cooldown windows after repeated failures.
4. Manual override actions logged with actor identity and reason.

---

## 14. Offline/Bluetooth extension requirements (post-MVP)

1. Support offline proximity intent exchange (BLE/NFC simulation layer).
2. Enforce strict per-txn/per-wallet limits.
3. Use expiring signed offline tokens.
4. Perform online sync reconciliation and double-spend conflict policy.
5. Mark offline accepted transactions as deferred-finality until sync completion.

---

## 15. Admin and ops requirements

1. View transaction timeline with state/event/ledger links.
2. View reversal queue and statuses.
3. Trigger reconciliation run manually.
4. View mismatch incidents and resolution state.
5. Export audit trail for selected transaction ids.

---

## 16. Acceptance criteria

Product is accepted when all conditions below are true:

1. Duplicate payment POST requests with same idempotency key do not create duplicate financial effects.
2. Debit-success/credit-failure path always leads to deterministic reversal workflow.
3. Every terminal transaction has complete event history and ledger evidence.
4. Reconciliation job produces mismatch report with classified reasons.
5. Observability dashboard can identify stuck transactions and backlog growth.
6. Load/concurrency tests show zero ledger invariant breaches.

---

## 17. Milestones (execution plan)

1. **M1 Foundation**: scaffolding, migrations, base entities, config, health endpoints.
2. **M2 Payment Core**: initiate/status APIs + idempotency records.
3. **M3 Ledger Core**: double-entry posting + invariants.
4. **M4 Async Core**: outbox relay + Kafka workers + orchestration state machine.
5. **M5 Failure Handling**: retries, reversal engine, DLQ strategy.
6. **M6 Reconciliation**: run pipeline + mismatch classification + admin views.
7. **M7 Hardening**: metrics, tracing, alerts, load/stress tests.
8. **M8 Release Candidate**: docs completion, demo scenarios, benchmark report.

---

## 18. Dependencies

1. PostgreSQL
2. Redis
3. Kafka
4. FastAPI runtime
5. Worker runtime (Celery/custom consumer)
6. Container orchestration for local/staging (Docker Compose minimum)

---

## 19. Risks and mitigations

1. **State machine drift**  
   Mitigation: centralized transition guard module + contract tests.

2. **Event/DB inconsistency**  
   Mitigation: transactional outbox mandatory.

3. **Silent duplicate effects**  
   Mitigation: idempotency at API + consumer + ledger write guard.

4. **Operational blind spots**  
   Mitigation: mandatory metrics/logging/tracing before release candidate.

5. **Scope creep**  
   Mitigation: keep offline/Bluetooth and AI enhancements post-MVP.

---

## 20. Success metrics

1. 100% invariant pass rate in integration/load suites.
2. 0 unresolved critical reconciliation incidents in test runs.
3. p95 payment-create latency target met.
4. Deterministic duplicate-request behavior verified in automated tests.
5. Complete architecture-to-implementation traceability across docs and code.

---

## 21. Out-of-scope policy note

Any claim or implementation that implies this simulator is a replacement for national payment infrastructure is out of scope.  
The system is explicitly educational/engineering-focused and should always be labeled as a simulator in docs and UI.

