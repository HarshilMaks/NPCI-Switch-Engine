# Test Strategy

## 1. Test layers
1. **Unit**: transition guards, invariant validators, idempotency logic.
2. **Integration**: API + DB + outbox + worker path.
3. **Failure injection**: retries, crash recovery, partial-step failures.
4. **Load/concurrency**: duplicate storms and high parallel payment requests.

## 2. Critical invariant test matrix
1. Duplicate request with same idempotency key -> one financial effect.
2. Illegal state transition -> rejected with reason code.
3. Debit success + credit failure -> reversal path reaches terminal state.
4. Completed/reversed transaction -> balanced debit/credit totals.
5. Worker replay after crash -> no duplicate ledger postings.

## 3. Reconciliation tests
1. Synthetic missing leg.
2. Amount mismatch.
3. Duplicate posting detection.
4. Stale pending detection.

## 4. Performance tests
1. API latency benchmarks (p50/p95/p99).
2. End-to-end completion latency under normal load.
3. Queue lag behavior under burst load.

## 5. Exit criteria
1. No critical invariant failures.
2. Deterministic outcomes across repeated runs.
3. Failure-injection scenarios produce expected compensation behavior.

