# Test Strategy

## 0. Test setup

### Running tests

```bash
# Unit tests
go test ./internal/...

# With coverage
go test -cover ./internal/...

# Verbose
go test -v ./internal/services
```

### Test database

Tests use SQLite in-memory (`:memory:`) for isolation and speed.

---

## 1. Test layers

1. **Unit**: payment service methods, state transitions, idempotency logic, decimal arithmetic.
2. **Integration**: API endpoints + DB + reconciliation flow (in-memory SQLite).
3. **Smoke tests**: CLI tool to verify critical paths (create → status → reconciliation).
4. **Load/concurrency**: `go-benchmark` for duplicate idempotency storms.

---

## 2. Critical invariant test matrix

| Test | Expected | Implementation |
|------|----------|-----------------|
| Duplicate idempotency key → one financial effect | PASSED (two requests return same txn_id) | `TestCreatePaymentIdempotency()` |
| Illegal state transition → rejected | PASSED (state guard validates) | `TestInvalidStateTransition()` |
| Debit success + credit fail → reversal reaches terminal | PENDING (phase 2: async reversal) | `TestAutoReversalPath()` |
| Completed/reversed → balanced ledger | PASSED (insert enforces dual posting) | `TestLedgerInvariant()` |
| Worker replay → no duplicate ledger | PENDING (phase 2: outbox relay) | `TestWorkerIdempotency()` |

---

## 3. Reconciliation tests

1. **Synthetic missing leg**: Manually corrupt ledger, verify detection.
2. **Amount mismatch**: Insert unbalanced pairs, verify flag.
3. **Duplicate posting**: Test dedup on reconciliation run (idempotent reruns).
4. **Stale pending**: Transaction stuck >10min in non-terminal state.

---

## 4. Performance targets

| Metric | Target | Method |
|--------|--------|--------|
| API latency (p50) | <50ms | `go test -bench BenchmarkCreatePayment` |
| API latency (p95) | <200ms | Simulated load test |
| End-to-end (INITIATED→COMPLETED) | <100ms | Sync orchestration timing |
| Reconciliation run (1k txns) | <5s | Bulk query performance |

---

## 5. Exit criteria

- ✓ No critical invariant failures in test matrix.
- ✓ Deterministic outcomes across repeated runs (idempotency verified).
- ✓ Failure-injection scenarios produce expected compensation (phase 2).
- ✓ All locked test cases pass locally before merge.

---

## 6. Test utilities

### Mock interfaces (phase 2)

```go
// Mock bank adapter for failed credit scenarios
type MockBankAdapter struct {
  ShouldFailCredit bool
}

func (m *MockBankAdapter) PostCredit(ctx context.Context, vpa string, amount decimal.Decimal) error {
  if m.ShouldFailCredit {
    return errors.New("downstream timeout")
  }
  return nil
}
```

### Fixture data

```go
const (
  PayerVPA   = "alice@bank"
  PayeeVPA   = "bob@bank"
  ValidAmount = "100.00"
)

func seedTestDB(t *testing.T, db *sql.DB) {
  // Setup 2 accounts + VPAs
}
```

