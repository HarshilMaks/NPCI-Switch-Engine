# NPCI/UPI System Architecture, Transaction Flow, and High-Grade Backend Simulator Blueprint

## 1. Scope and intent

This document consolidates:

1. A practical architecture view of NPCI/UPI-style systems.
2. End-to-end transaction flow details.
3. How integration with banks, external systems, and card networks works at a high level.
4. A reality check of common claims/speculation.
5. A production-grade blueprint for building a **UPI-inspired payment rail simulator backend**.
6. “Revolutionary” extension ideas (including Bluetooth/offline payments) with guardrails.
7. Useful research and competition directions.

> Important: This is an educational/system-design reference, **not** an official NPCI specification and **not** a real-money deployment guide.

---

## 2. NPCI/UPI architecture: practical mental model

At system level, think in 5 domains:

1. **Client/TPAP apps**  
   User-facing apps (bank apps + third-party apps) that initiate intents and present status.

2. **PSP bank layer**  
   Handles VPA-linked identity mapping, risk controls, authentication orchestration, request shaping, retries, and customer notifications.

3. **NPCI UPI switch domain**  
   Interoperability switch for routing, scheme-rule enforcement, participant addressing, central message exchange, and scheme-level lifecycle handling.

4. **Issuer and Acquirer bank systems**  
   Core account checks, debit/credit posting, account restrictions, limits, and authorization decisions.

5. **Settlement, reconciliation, and dispute domain**  
   Interbank settlement cycles, reconciliation files, exception handling, reversals, and formal dispute workflows.

### Key architecture properties

- Interoperability-first design.
- Strong reliability controls (idempotency, retries, timeout handling).
- Real-time customer response path with non-trivial post-processing path (reconciliation/disputes).
- Multi-party trust boundaries with strict cryptographic/authentication controls.

---

## 3. UPI-style transaction flow (high-fidelity sequence)

### Phase A: initiation and customer authentication

1. User initiates payment/collect in app (VPA/QR/intent).
2. App + PSP layer perform device/app/session checks.
3. Customer authentication/authorization is captured (e.g., PIN flow).
4. PSP constructs signed request with idempotency and trace identifiers.

### Phase B: switching and routing

5. PSP forwards request to central switching domain.
6. Switch resolves destination participant/bank from addressing data (e.g., VPA resolution context).
7. Switch validates scheme-level fields and routes to remitter/beneficiary participants.

### Phase C: bank-side posting

8. Payer-side bank validates limits/balance/status and authorizes debit path.
9. Payee-side bank validates beneficiary state and processes credit path.
10. Result statuses propagate back through switch to both participant sides.

### Phase D: completion and aftercare

11. Payer and payee apps receive final customer-visible status.
12. Asynchronous downstream: reconciliation artifacts, exception queues, reversal triggers, and dispute operations.

---

## 4. How connections to external systems work

### 4.1 PSP/bank connectivity

- Strongly governed participant onboarding.
- Standardized message formats and scheme-defined operational contracts.
- Certificate-based trust, secure transport, and strict operational SLAs.

### 4.2 Card network context (RuPay and others)

- Card rails and UPI rails are related ecosystem components but not identical transaction mechanics.
- Card authorization/clearing/settlement flows differ from account-to-account push flows.
- Any “single protocol everywhere” claim is usually an oversimplification.

### 4.3 Adjacent national payment systems

- IMPS, AEPS, NACH, BBPS, etc. are separate schemes with distinct business/process rules.
- Organization-level governance may be shared; transaction semantics are not always interchangeable.

---

## 5. Security and reliability fundamentals

Minimum baseline expected in payment-grade systems:

1. **Cryptographic trust controls**: strong transport security + participant certificate controls.
2. **Idempotency**: request and message-level deduplication.
3. **Atomic money movement rules**: no “half-success” without compensating path.
4. **Immutable audit trail**: append-only events and forensic traceability.
5. **Fraud/risk controls**: velocity, anomaly, and participant behavior policies.
6. **Timeout + retry governance**: bounded retries and deterministic conflict handling.

---

## 6. Reality check on common architecture claims

### Usually valid direction

- Layered model: client/PSP/switch/bank/settlement.
- Real-time user-facing payment confirmation flow.
- High emphasis on reliability and anti-fraud controls.

### Commonly overstated or speculative

- Exact infrastructure node counts and internal server topology without official disclosures.
- Simplified “instant everything including settlement” claims.
- Protocol assumptions transferred from card systems to UPI without qualification.

---

## 7. Backend project: production-grade UPI-inspired simulator

## 7.1 What this project is

A **high-fidelity payment rail simulator** that teaches and demonstrates:

- Double-entry accounting integrity
- Distributed transaction orchestration
- Idempotent processing
- Failure-safe compensation (reversal/refund)
- Reconciliation and audit rigor

## 7.2 What this project is not

- Not an official NPCI stack.
- Not integrated with live banking rails.
- Not for real-money production use.

---

## 8. Recommended technical stack (backend-first)

- **Python**
- **FastAPI**
- **PostgreSQL** (strict ACID)
- **Redis** (idempotency cache and fast state references)
- **Celery** (scheduled/async jobs)
- **Kafka** (event stream/outbox publication target)
- **SQLAlchemy**
- **Docker Compose**
- Optional admin UI (React or equivalent)

---

## 9. Core domain design for simulator

### 9.1 Ledger-first principle

- Every financial movement represented as balanced debit/credit entries.
- Append-only ledger events; avoid mutable accounting history.

### 9.2 Transaction state machine

Example lifecycle:

`INITIATED -> AUTH_PENDING -> AUTHORIZED -> POSTED -> COMPLETED`  
Failure paths:
`... -> FAILED` or `... -> REVERSAL_PENDING -> REVERSED`

Each transition must log:

- from_state
- to_state
- reason_code
- actor/system source
- timestamp

### 9.3 Idempotency model

- Mandatory idempotency key at API boundary.
- Persist idempotency fingerprint and deterministic response payload.
- Ensure safe retries under network interruptions.

### 9.4 Reversal and compensation

- If debit succeeds but downstream completion fails, generate compensating reversal transaction.
- Reversal itself is a first-class transaction with trace links to original txn.

### 9.5 Reconciliation engine

- Compare internal ledger postings against settlement-like logs.
- Detect mismatch classes:
  - missing leg
  - amount mismatch
  - duplicate posting
  - stale pending
- Emit daily and intraday reports.

### 9.6 Risk controls

- Velocity checks (txn/min, txn/hour).
- Amount anomaly flags.
- Merchant/user risk tiers.
- Dynamic throttles and manual review queues.

---

## 10. Suggested schema (baseline)

Core tables (illustrative):

- `accounts(id, user_id, currency, status, created_at)`
- `account_balances(account_id, available, blocked, updated_at)` *(or derive via ledger + snapshots)*
- `transactions(id, payer_vpa, payee_vpa, amount, currency, status, idempotency_key, created_at, completed_at)`
- `transaction_events(id, transaction_id, from_status, to_status, reason_code, metadata, created_at)`
- `ledger_entries(id, transaction_id, account_id, leg_type, amount, currency, created_at)` *(append-only)*
- `reversals(id, original_transaction_id, reversal_transaction_id, reason, status, created_at)`
- `reconciliation_runs(id, run_date, status, summary, created_at)`
- `reconciliation_diffs(id, run_id, transaction_id, diff_type, details, created_at)`

---

## 11. APIs and event contracts (recommended)

### API surfaces

1. `POST /payments` (idempotent create/initiate)
2. `GET /payments/{id}` (state query)
3. `POST /payments/{id}/confirm` (auth confirm simulation)
4. `POST /payments/{id}/cancel`
5. `POST /reversals`
6. `POST /reconciliation/runs`
7. `GET /ledger/accounts/{id}/entries`

### Event topics (example)

- `payment.initiated`
- `payment.authorized`
- `payment.posted`
- `payment.completed`
- `payment.failed`
- `payment.reversal.requested`
- `payment.reversed`
- `reconciliation.diff.detected`

---

## 12. Revolutionary extension: Bluetooth/offline payments

This is feasible as a **bounded-risk deferred settlement** mode.

### 12.1 Feasible architecture concept

- Proximity transfer over BLE/NFC of signed payment intents/tokens.
- Receiver stores proof package.
- Final ledger posting occurs on next connectivity sync.

### 12.2 Main risk: offline double spend

Mitigation set:

1. Very strict per-transaction and wallet/device caps.
2. Token expiry windows.
3. One-time spend proofs with monotonic counters.
4. Device-bound cryptographic keys (secure enclave/TEE where possible).
5. Risk reserve and delayed merchant finality until sync.

### 12.3 Compliance alignment hint

- RBI has an offline small-value framework with bounded limits and explicit consent requirements.  
  Use this as policy inspiration in simulator mode.

---

## 13. Guardrails for optional AI in sensitive backend

AI can be used for:

- Fraud scoring assistance
- Alert prioritization
- Operator copilots for reconciliation triage

AI should **not** directly control:

- Final debit/credit posting logic
- Ledger mutation
- Unsupervised dispute outcomes

Guardrails:

1. Deterministic rule engine remains source of truth.
2. Human approval for high-risk actions.
3. Full model decision logging with explainability snapshots.
4. Strict PII minimization and access controls.

---

## 14. Research and competition directions

1. **NPCI UPI circulars/resources** for evolving scheme guidance.
2. **RBI offline digital payments framework** (including updates that raised UPI Lite limits in that framework context).
3. **National Fintech Hackathon 2026** (Ericsson + FIRST IIT Kanpur) as a relevant innovation arena.
4. Broader RBI/RBIH challenge programs and sandbox themes for payment inclusion/security.

---

## 15. Build outcome statement (resume-ready)

“Built a production-grade UPI-inspired payment rail simulator with append-only double-entry ledger, deterministic transaction state machine, idempotent processing, compensating reversals, and automated reconciliation. Designed for failure recovery, auditability, and high-concurrency correctness in distributed backend systems.”

