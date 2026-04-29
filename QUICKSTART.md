# Quick Start — NPCI UPI Payment Simulator

Get the payment simulator running in 2 minutes.

## 1. Build

```bash
cd npci-upi
go mod download
go build -o ./bin/server ./cmd/server
```

**Output**: `./bin/server` (15MB, fully static, zero external dependencies)

## 2. Run

```bash
./bin/server
```

**Output**:
```
🚀 Payment Switch Engine starting on :8080
```

Server ready on `http://localhost:8080`

## 3. Test

In a new terminal:

```bash
# Health check
curl http://localhost:8080/health
# {"status":"healthy"}

# Create payment (idempotent)
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-001" \
  -d '{
    "payer_vpa": "alice@bank",
    "payee_vpa": "bob@bank",
    "amount": "100.00",
    "currency": "INR"
  }'

# Get payment status
curl http://localhost:8080/api/v1/payments/{transaction_id}

# Run reconciliation
curl -X POST http://localhost:8080/api/v1/reconciliation/run
```

## 4. Explore

- **README.md** — Full feature list and API documentation
- **.lock/runbook.md** — Operations and troubleshooting guide
- **.lock/architecture.md** — System design and data flow
- **.lock/tech-stack.md** — Why Go, tech choices, project layout

## Next steps

- Read `.lock/prd.md` for requirements
- Review `.lock/implementation-plan.md` for phase status
- Check `.lock/test-strategy.md` for critical paths
- See `.lock/runbook.md` for operational procedures

---

**Seeded accounts** (ready to use):
- `alice@bank` — 1M INR
- `bob@bank` — 1M INR
- `inactive@bank` — 0 INR (for testing reversals)

**Database**: SQLite (auto-created as `npci_upi.db`)

---

Questions? See the runbook or README for more details.
