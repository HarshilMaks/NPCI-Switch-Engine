# Operations Runbook

## 0. Local development setup

### Build from source

```bash
cd npci-upi
go mod download
go build -o ./bin/server ./cmd/server
```

### Run locally

```bash
# Start server (creates/uses npci_upi.db)
DATABASE_URL="file:npci_upi.db?_pragma=busy_timeout(5000)" \
DEFAULT_SEED_BALANCE="1000000.00" \
./bin/server

# Server listens on http://localhost:8080
```

### Verify health

```bash
curl http://localhost:8080/health
# Expected: {"status":"healthy"}
```

### Test payment flow

```bash
# Create payment
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-001" \
  -d '{
    "payer_vpa": "alice@bank",
    "payee_vpa": "bob@bank",
    "amount": "100.00",
    "currency": "INR"
  }'

# Get status
curl http://localhost:8080/api/v1/payments/{transaction_id}

# Run reconciliation
curl -X POST http://localhost:8080/api/v1/reconciliation/run
```

### Configuration

All settings via environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | `file:npci_upi.db?_pragma=busy_timeout(5000)` | SQLite connection string |
| `DEFAULT_CURRENCY` | `INR` | Default transaction currency |
| `DEFAULT_SEED_BALANCE` | `1000000.00` | Initial account balance for seeded VPAs |
| `HOLDING_ACCOUNT_ID` | `system-holding-account` | System holding account for reversals |

---

## 1. Stuck transaction (non-terminal too long)

1. Locate transaction by ID in database: `SELECT * FROM transactions WHERE id = ?`
2. Check latest event: `SELECT * FROM transaction_events WHERE transaction_id = ? ORDER BY created_at DESC LIMIT 1`
3. Inspect transaction_events for reason code (AUTH_FAILED, INSUFFICIENT_FUNDS, PAYEE_UNAVAILABLE, etc.)
4. If DEBIT_POSTED and credit failed: reversal should be REVERSED or REVERSAL_PENDING.
5. Check `reversals` table: `SELECT * FROM reversals WHERE original_transaction_id = ?`
6. If no reversal exists but debit posted, manually trigger via: `POST /api/v1/reversals`

---

## 2. Reconciliation mismatch spike

1. Verify latest reconciliation run: `SELECT * FROM reconciliation_runs ORDER BY completed_at DESC LIMIT 1`
2. Fetch diffs: `SELECT diff_type, COUNT(*) FROM reconciliation_diffs WHERE run_id = ? GROUP BY diff_type`
3. Inspect diff types:
   - `missing_leg`: Transaction has no debit or no credit (invariant breach)
   - `amount_mismatch`: Debit total != credit total
   - `stale_pending`: Transaction stuck in non-terminal state >10min
4. Re-run reconciliation: `POST /api/v1/reconciliation/run`
5. If mismatch persists: escalate as SEV-1 (financial inconsistency risk)

---

## 3. High reversal ratio

1. Query reversal statistics: `SELECT COUNT(*) as total, status FROM reversals GROUP BY status`
2. Check reason codes: `SELECT reason_code, COUNT(*) FROM transaction_events WHERE to_status = 'REVERSED' GROUP BY reason_code`
3. Dominant codes:
   - `PAYEE_UNAVAILABLE`: Downstream adapter is down (check bank simulator)
   - `INSUFFICIENT_FUNDS`: System holding account depleted (reseed)
   - `AUTH_FAILED`: Risk check failure (review recent policy changes)
4. Escalate if reversal ratio > 5% of total transactions (configure threshold)

---

## 4. Outbox backlog growth

1. Check unpublished events: `SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL`
2. If > 100, check server logs for relay errors
3. In Phase 2 (async workers): scale outbox relay pool
4. For now (Phase 1), outbox is prepared but not yet consumed

---

## 5. Severity model (minimum)

1. **SEV-1**: Invariant breach (missing_leg or amount_mismatch detected).
   - Action: Halt payments, investigate immediately
2. **SEV-2**: Sustained transaction completion degradation (>50% failures in 1min window).
   - Action: Escalate to risk, review recent changes
3. **SEV-3**: Non-critical subsystem impairment (reconciliation delayed, logs noisy).
   - Action: Schedule review, no immediate action required

---

## 6. Emergency procedures

### Database corruption (unlikely but catastrophic)

1. Snapshot current database: `cp npci_upi.db npci_upi.db.backup`
2. Run full reconciliation: `POST /api/v1/reconciliation/run`
3. Inspect reconciliation_runs summary for invariant breaches
4. If confirmed: escalate to incident response team

### Reset for testing

```bash
# Stop server
rm npci_upi.db
./bin/server  # Recreates with fresh schema + seed data
```

