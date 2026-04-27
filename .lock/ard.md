# ARD — Architecture Requirements Document  
## UPI-Inspired Payment Rail Simulator (Backend-First)

## 1. Document control

- **Document**: Architecture Requirements Document (ARD)
- **Version**: 1.0
- **Status**: Baseline
- **Scope**: Backend simulator architecture and implementation constraints
- **References**:
  - `.lock/prd.md`
  - `.lock/architecture.md`
  - `.lock/npci-upi-system-architecture-and-simulator-blueprint.md`

---

## 2. Purpose

Define the architecture-level requirements that implementation must satisfy to ensure:

1. financial correctness
2. deterministic distributed behavior
3. immutable auditability
4. production-style reliability and operability

---

## 3. Architecture principles (mandatory)

1. **Ledger-first correctness**: monetary truth comes from immutable double-entry records.
2. **Deterministic state progression**: all transaction transitions must be explicit and validated.
3. **Idempotency by default**: duplicate requests/events must not cause duplicate financial effects.
4. **Failure compensation over silent rollback**: compensating transactions are first-class.
5. **Observability as a built-in requirement**: every critical path must be measurable and traceable.

---

## 4. System context and boundaries

## 4.1 In-scope systems

1. API Gateway + Payment API
2. VPA Directory service
3. Auth/Risk service
4. Orchestrator worker(s)
5. Bank simulator adapters (payer/payee)
6. Reconciliation service
7. Admin/ops API
8. Event bus + outbox relay

## 4.2 Out-of-scope systems

1. Real bank/NPCI connectivity
2. Real card network protocols
3. Production regulatory adjudication flows

---

## 5. Logical architecture requirements

## 5.1 Service decomposition

The system shall be split into bounded contexts:

1. Payments
2. Ledger
3. Orchestration
4. Reconciliation
5. Risk/controls
6. Admin/ops

Each context shall expose a clear API/contract and avoid cross-context direct DB mutation.

## 5.2 Data ownership

1. Payments context owns transaction aggregate and lifecycle state.
2. Ledger context owns immutable postings and invariant enforcement.
3. Reconciliation context owns run artifacts and mismatch classification.
4. Shared reads are allowed; cross-context writes are prohibited except through contracts.

---

## 6. State machine requirements

## 6.1 Allowed states

- `INITIATED`
- `AUTH_PENDING`
- `AUTHORIZED`
- `DEBIT_POSTED`
- `CREDIT_POSTED`
- `COMPLETED`
- `FAILED`
- `REVERSAL_PENDING`
- `REVERSED`
- `REVERSAL_FAILED`

## 6.2 Transition enforcement

1. Transition matrix shall be centrally defined and versioned.
2. Illegal transitions must fail with explicit reason code.
3. Every state transition shall append one event row with actor, reason, timestamp.
4. Terminal states shall be immutable.

---

## 7. Ledger requirements

## 7.1 Posting model

1. Every monetary mutation shall map to balanced debit/credit entries.
2. Ledger entries shall be append-only.
3. Updates/deletes on ledger rows are forbidden at application level.

## 7.2 Invariant rules

1. `sum(debits) == sum(credits)` per settled or reversed transaction.
2. No transaction can be terminal without required ledger legs.
3. Reversal postings must reference original transaction linkage.

---

## 8. API architecture requirements

## 8.1 Public API requirements

Required endpoints:

1. `POST /api/v1/payments`
2. `GET /api/v1/payments/{transaction_id}`
3. `POST /api/v1/payments/{transaction_id}/confirm`
4. `POST /api/v1/payments/{transaction_id}/cancel`
5. `POST /api/v1/reversals`
6. `POST /api/v1/reconciliation/run`
7. `GET /api/v1/accounts/{account_id}/ledger`

## 8.2 API contract requirements

1. Mutation endpoints shall require `Idempotency-Key`.
2. `Correlation-Id` shall be accepted/generated and propagated.
3. Error response shall follow one canonical machine-readable schema.
4. API versions shall be path-versioned (`/api/v1/...`).

---

## 9. Messaging and outbox requirements

## 9.1 Outbox pattern

1. Domain write + outbox write shall occur in one DB transaction.
2. Outbox relay shall provide at-least-once publishing to event bus.
3. Outbox rows shall include event id, aggregate id, schema version, created_at.

## 9.2 Consumer behavior

1. Consumers shall be idempotent per event id.
2. Retriable failures shall backoff with jitter.
3. Poison messages shall route to DLQ with diagnostic metadata.

---

## 10. Storage architecture requirements

## 10.1 PostgreSQL requirements

1. Primary source of truth for transactions, events, ledger, and outbox.
2. ACID semantics for write paths.
3. Indexed access paths for transaction status, idempotency keys, and event replay.

## 10.2 Redis requirements

1. Fast idempotency reservation/check.
2. Short-lived distributed lock primitives (where required).
3. Rate-limit counters for risk controls.

## 10.3 Kafka/event bus requirements

1. Topic partitioning by transaction/account key.
2. Event schema versioning policy.
3. Retention sufficient for replay/debug windows.

---

## 11. Reliability and fault-tolerance requirements

1. Worker restart/reprocessing shall not violate monetary invariants.
2. Duplicate API calls shall never produce duplicate financial effect.
3. Debit-success/credit-failure path shall deterministically trigger reversal.
4. Stuck non-terminal transactions beyond SLA shall be detectable and alertable.
5. Background jobs shall be resumable and checkpoint-safe.

---

## 12. Reconciliation requirements

1. Reconciliation shall support scheduled and on-demand runs.
2. Run output shall classify mismatch types:
   - `missing_leg`
   - `amount_mismatch`
   - `duplicate_posting`
   - `stale_pending`
3. Reconciliation artifacts shall be immutable and queryable.
4. Incident records shall support lifecycle statuses (`OPEN`, `INVESTIGATING`, `RESOLVED`).

---

## 13. Security architecture requirements

1. TLS mandatory for all service traffic.
2. Service identity/auth between internal components.
3. Secrets shall not be stored in repository.
4. Sensitive identifiers shall be masked/tokenized in logs.
5. Admin endpoints shall enforce RBAC and audit trails.

---

## 14. Observability requirements

## 14.1 Logging

1. Structured logs (JSON).
2. Required fields: `timestamp`, `service`, `level`, `transaction_id`, `correlation_id`, `event_id`.

## 14.2 Metrics

1. API p50/p95/p99 latency.
2. Transaction completion latency.
3. Reversal count and ratio.
4. Reconciliation mismatch count.
5. Queue lag/backlog metrics.

## 14.3 Tracing

1. Distributed trace continuity from API to worker/adapters.
2. Span annotations for state transitions and external adapter calls.

---

## 15. Performance and scalability requirements

1. System shall scale API tier horizontally (stateless services).
2. Worker scale-out shall be independent from API tier.
3. Partition strategy shall avoid hot keys under realistic load.
4. Read-heavy ops/admin queries should not block critical write path.

Target baselines (simulator):

1. Payment initiate API p95 < 150 ms (normal load).
2. Happy path async completion < 2 seconds target.

---

## 16. Deployment and environment requirements

1. Local environment shall run via Docker Compose.
2. Staging shall mirror production-sim component topology.
3. All deployable services shall have health and readiness endpoints.
4. Schema migration workflow shall be deterministic and reversible where safe.

---

## 17. Compliance and labeling requirements

1. All docs and UI surfaces shall clearly label the system as a simulator/sandbox.
2. No feature shall imply official payment network replacement capability.

---

## 18. Testing and validation requirements

## 18.1 Required test classes

1. Unit tests for transition guards and invariant checks.
2. Integration tests for API + DB + outbox.
3. Worker replay/idempotency tests.
4. Fault-injection tests for timeout/crash/retry paths.
5. Concurrency/load tests for invariant safety.

## 18.2 Critical test assertions

1. No duplicate ledger effect under repeated request replay.
2. No illegal transition accepted.
3. Reversal path always links to original transaction and reaches terminal status.
4. Reconciliation correctly reports synthetic mismatch datasets.

---

## 19. ADR (architecture decision record) requirements

The project shall maintain ADRs for:

1. storage source-of-truth choice
2. outbox strategy
3. orchestration pattern (saga)
4. idempotency implementation
5. reconciliation strategy

Each ADR must include context, decision, alternatives, and consequences.

---

## 20. Exit criteria (architecture sign-off)

Architecture implementation is considered complete when:

1. all mandatory requirements in this ARD are implemented or explicitly waived
2. financial invariants pass under integration and load scenarios
3. observability dashboards expose end-to-end transaction health
4. incident/reconciliation workflows are operationally usable
5. PRD acceptance criteria are demonstrably satisfied

