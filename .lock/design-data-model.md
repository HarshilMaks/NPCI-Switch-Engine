# Design Doc 02 — Data Model and Invariants

## 1. Data stores

1. **PostgreSQL**: source of truth.
2. **Redis**: idempotency and rate controls.
3. **Kafka**: asynchronous event transport.

## 2. Logical schema

## 2.1 Core entities

- `accounts(id, user_id, currency, status, created_at)`
- `vpas(id, handle, account_id, status, created_at)`
- `transactions(id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at)`
- `transaction_events(id, transaction_id, from_status, to_status, reason_code, actor, metadata, created_at)`
- `ledger_entries(id, transaction_id, account_id, leg_type, amount, currency, created_at)`  *(append-only)*
- `reversals(id, original_transaction_id, reversal_transaction_id, reason, status, created_at)`
- `idempotency_records(idempotency_key, scope_key, request_hash, response_payload, status_code, created_at, expires_at)`
- `outbox_events(id, aggregate_type, aggregate_id, event_type, payload, published_at, created_at)`
- `reconciliation_runs(id, run_key, started_at, completed_at, status, summary_json)`
- `reconciliation_diffs(id, run_id, transaction_id, diff_type, details_json, created_at)`

## 2.2 Suggested indexes

1. `transactions(status, created_at)`
2. `transactions(idempotency_key)`
3. `transaction_events(transaction_id, created_at)`
4. `ledger_entries(transaction_id, created_at)`
5. `outbox_events(published_at, created_at)`
6. `reconciliation_diffs(run_id, diff_type)`

## 3. Invariants (hard requirements)

1. For each completed or reversed flow: `sum(debit) = sum(credit)`.
2. Ledger rows are immutable after insert.
3. Idempotency key in same scope maps to one canonical response.
4. Transaction `version` increments on each valid mutation.
5. Terminal states cannot transition further.

## 4. Concurrency strategy

1. Use optimistic locking (`version`) for transaction aggregate updates.
2. Use unique constraints on idempotency scope keys.
3. Keep posting operations in DB transactions to guarantee atomic ledger writes.

## 5. Retention strategy

1. Ledger and transaction events retained long-term (audit-first).
2. Outbox rows retained with archival process after publish confirmation.
3. Idempotency records TTL-based cleanup with configurable retention.

