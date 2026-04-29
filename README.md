# NPCI UPI Payment Simulator (Backend) — Go Implementation

UPI-inspired payment rail simulator focused on financial correctness and auditability.

## What's inside

- **Double-entry ledger**: Every payment posts debit/credit pairs; invariant enforced
- **State machine**: Deterministic transaction lifecycle (INITIATED → AUTHORIZED → DEBIT_POSTED → CREDIT_POSTED → COMPLETED)
- **Idempotency**: Duplicate requests return cached response (request hash + scope key)
- **Automatic reversal**: Failed credit automatically triggers compensating debit
- **Manual reversal**: Explicit reversal endpoint for operational needs
- **Reconciliation**: Auditable mismatch detection (missing legs, amount mismatches, stale pending)

**⚠️ Simulator only**: Not connected to real NPCI/UPI; useful for learning payment systems design.

---

## Quick start

### Prerequisites
- Go 1.22+
- `make` (optional, for convenience)

### Build and run

```bash
# Build
go mod download
go build -o ./bin/server ./cmd/server

# Run (creates npci_upi.db on first run)
./bin/server
# Server listening on http://localhost:8080
```

### Environment variables

```bash
DATABASE_URL="file:npci_upi.db?_pragma=busy_timeout(5000)"  # SQLite connection
DEFAULT_CURRENCY="INR"                                       # Default currency
DEFAULT_SEED_BALANCE="1000000.00"                           # Seed account balance
HOLDING_ACCOUNT_ID="system-holding-account"                 # System holding account
```

---

## API Overview

### Create payment (idempotent)

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-001" \
  -d '{
    "payer_vpa": "alice@bank",
    "payee_vpa": "bob@bank",
    "amount": "100.00",
    "currency": "INR"
  }'

# Response
{
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "accepted_at": "2026-04-29T04:30:00Z"
}
```

### Get payment status

```bash
curl http://localhost:8080/api/v1/payments/{transaction_id}

# Response
{
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "amount": "100.00",
  "currency": "INR",
  "events": [
    {"from_status":"INITIATED","to_status":"AUTH_PENDING","reason_code":"AUTH_REQUESTED"},
    {"from_status":"AUTH_PENDING","to_status":"AUTHORIZED","reason_code":"AUTH_APPROVED"},
    ...
  ]
}
```

### Manual reversal

```bash
curl -X POST http://localhost:8080/api/v1/reversals \
  -H "Content-Type: application/json" \
  -d '{
    "original_transaction_id": "550e8400-e29b-41d4-a716-446655440000",
    "reason": "customer_request"
  }'
```

### Run reconciliation

```bash
curl -X POST http://localhost:8080/api/v1/reconciliation/run

# Response
{
  "run_id": "550e8400-e29b-41d4-a716-446655440000",
  "run_key": "recon-20260429043000",
  "status": "COMPLETED",
  "summary": {
    "inspected_transactions": 42,
    "diff_count": 0,
    "amount_mismatch_count": 0,
    "stale_pending_count": 0,
    "unpublished_outbox_events": 8
  }
}
```

### Health check

```bash
curl http://localhost:8080/health
# {"status":"healthy"}
```

---

## Seeded accounts

Out of the box, the database seeds:

| VPA | Account | Balance | Status |
|-----|---------|---------|--------|
| `alice@bank` | alice-account | 1M INR | ACTIVE |
| `bob@bank` | bob-account | 1M INR | ACTIVE |
| `inactive@bank` | inactive-merchant-account | 0 | INACTIVE |
| `system@bank` | system-holding-account | 0 | ACTIVE |

---

## Architecture

### Tech stack

- **Language**: Go 1.22
- **HTTP Router**: `github.com/go-chi/chi/v5`
- **Database**: SQLite (modernc.org/sqlite, pure Go)
- **Decimal Math**: `github.com/shopspring/decimal` (exact financial arithmetic)

### Project layout

```
npci-upi/
├── cmd/server/
│   └── main.go                     # Server entrypoint
├── internal/
│   ├── config/config.go            # Config loader
│   ├── storage/
│   │   ├── db.go                   # DB connection
│   │   ├── migrate.go              # Schema creation
│   │   └── seed.go                 # Bootstrap data
│   ├── types/types.go              # Request/response types
│   ├── state/state.go              # State machine
│   ├── services/
│   │   ├── errors.go               # Error handling
│   │   ├── payment.go              # Payment orchestration (~650 lines)
│   │   └── reconciliation.go       # Ledger verification
│   └── handlers/payment.go         # HTTP routing
├── go.mod                          # Dependencies
├── go.sum                          # Checksums
├── bin/server                      # Compiled binary (after build)
└── .lock/                          # Immutable reference docs
```

### Key invariants

1. **Ledger balance**: For every COMPLETED transaction, sum(debits) == sum(credits).
2. **Idempotency**: Same request + same idempotency key = same response (not replayed).
3. **State determinism**: Transition from state X to state Y is always valid or always rejected.
4. **Audit trail**: Every state change recorded in transaction_events (immutable).

---

## Documentation

All architectural decisions and operational procedures are locked in `.lock/`:

- **`.lock/prd.md`**: Business requirements
- **`.lock/ard.md`**: Architecture requirements
- **`.lock/architecture.md`**: System design and data flow
- **`.lock/tech-stack.md`**: Go/SQLite rationale and decisions
- **`.lock/implementation-plan.md`**: Phase completion status
- **`.lock/test-strategy.md`**: Testing approach and critical paths
- **`.lock/runbook.md`**: Operational procedures (setup, troubleshooting, incidents)
- **`.lock/api-spec.md`**: Full API contract
- **`.lock/security-threat-model.md`**: Security and compliance considerations

Start with **`.lock/npci-upi-system-architecture-and-simulator-blueprint.md`** for the full context.

---

## Development

### Run smoke tests

```bash
# Validates: create → status → idempotency → reconciliation
go run ./cmd/server ... &
# (Server runs in background, tests in foreground)
```

### Run tests (when available)

```bash
go test ./internal/...
```

### Build for distribution

```bash
go build -o ./bin/server ./cmd/server
# Output: single 20MB statically-linked binary
```

---

## Roadmap

**Phase 1 (MVP — ✓ COMPLETE)**
- Single payment flow (idempotent, orchestrated synchronously)
- Ledger with double-entry invariant
- Reconciliation runner

**Phase 2 (Async) — IN PROGRESS**
- Outbox relay worker (goroutines polling outbox_events)
- Event schema and consumer bootstrap
- Kafka/NATS integration preparation

**Phase 3+ (Scale)**
- Horizontal scaling (API replicas + event bus)
- Prometheus metrics and distributed tracing
- Multi-tenant isolation
- Payment corridors and routing

---

## License

This is an educational project. Not for production payment processing.

---

## Questions?

See `.lock/runbook.md` for common operational issues.  
See `.lock/design-overview.md` for deep architectural context.
