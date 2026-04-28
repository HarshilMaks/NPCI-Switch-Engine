# Tech Stack Decision — Go Implementation

**Date**: 2026-04-28  
**Status**: Locked (implementation baseline)

---

## Why Go?

Evaluated: Python/FastAPI, TypeScript/Node.js, Rust, Go

**Selected: Go** for payment systems backend.

### Decision rationale

1. **Concurrency model**: goroutines + channels are lightweight and idiomatic for payment event processing.
2. **Performance**: Native compilation to binary, sub-millisecond latencies, minimal GC pauses.
3. **Simplicity vs Rust**: Fast iteration, simpler syntax, no borrow checker friction for this domain.
4. **Better than TypeScript**: Strong static typing + runtime efficiency for financial correctness.
5. **Deployment**: Single binary, cross-platform builds, no runtime dependency hell.
6. **Learning curve**: Shorter onboarding for payment engineers than Rust; more robust than Python for production financial systems.

---

## Core dependencies

| Component | Package | Rationale |
|-----------|---------|-----------|
| **HTTP Router** | `github.com/go-chi/chi/v5` | Lightweight, idiomatic, minimal dependencies. |
| **Database Driver** | `modernc.org/sqlite` | Pure Go SQLite (no CGO), embedded, ACID transactions. |
| **Decimal Arithmetic** | `github.com/shopspring/decimal` | Exact financial arithmetic (no float rounding). |
| **UUID Generation** | `github.com/google/uuid` | Standard UUIDs for transaction IDs. |

---

## Database: SQLite

### Why SQLite (not PostgreSQL)?

| Aspect | SQLite | PostgreSQL |
|--------|--------|-----------|
| Deployment | Single file, embedded | Separate service, network I/O |
| ACID | Full ✓ | Full ✓ |
| Transactions | Single-file locking | Network-aware |
| MVP simplicity | Excellent (file-based) | Requires separate container |
| Scaling | WAL mode for concurrent reads | Built for distributed |

**MVP choice**: SQLite with WAL mode (Write-Ahead Logging) for single-node phase 1-2.

**Phase 3+ path**: Migrate to PostgreSQL with minimal schema changes (SQL is portable).

### Key schema characteristics

- **`transactions`**: Primary aggregate root; status + version for optimistic concurrency.
- **`ledger_entries`** (append-only): Core financial truth; no updates, only inserts.
- **`idempotency_records`**: Request hash + scope key = unique constraint for duplicate detection.
- **`outbox_events`**: Outbox pattern for reliable async event propagation.
- **`reconciliation_runs`**: Audit trail of ledger verification runs.

---

## Runtime architecture

### Single Go binary

```
cmd/server/main.go
  ├─ config.Load() → Settings
  ├─ storage.Open() → *sql.DB
  ├─ storage.Migrate() → Schema creation
  ├─ storage.Seed() → Bootstrap data
  ├─ services.NewPaymentService(db) → Business logic
  ├─ services.NewReconciliationService(db)
  ├─ handlers.NewPaymentHandler() → HTTP routing
  └─ chi.Router() on :8080
```

### Concurrency model

- **HTTP handlers**: One goroutine per request (chi middleware handles).
- **Transaction processing**: Synchronous within request (phase 1); async workers via goroutines (phase 2+).
- **Outbox relay** (future): Dedicated worker goroutine polling `outbox_events` table.

### No external dependencies for MVP

- ✓ No Redis (idempotency via DB)
- ✓ No Kafka (outbox table for async)
- ✓ No separate auth service (bearer tokens/internal only)

---

## Metrics and observability (phase 2+)

**Current**: Structured logging via standard `log` package.

**Phase 2 additions**:
- `github.com/prometheus/client_golang` → Prometheus metrics.
- `go.opentelemetry.io/...` → Distributed tracing (optional).
- ELK/Datadog agent → Log aggregation.

---

## Project structure

```
npci-upi/
├── cmd/server/
│   └── main.go                    # Entrypoint
├── internal/
│   ├── config/config.go           # Settings loading
│   ├── storage/
│   │   ├── db.go                  # DB connection
│   │   ├── migrate.go             # Schema creation
│   │   └── seed.go                # Bootstrap data
│   ├── types/types.go             # Request/response types
│   ├── state/state.go             # State machine validation
│   ├── services/
│   │   ├── errors.go              # AppError wrapper
│   │   ├── payment.go             # Payment orchestration
│   │   └── reconciliation.go      # Ledger verification
│   └── handlers/payment.go        # HTTP endpoints
├── go.mod                         # Dependency manifest
├── go.sum                         # Checksums
└── .lock/                         # Immutable reference docs
```

---

## Running locally

```bash
# Build
go mod download
go build -o ./bin/server ./cmd/server

# Run (creates/uses npci_upi.db)
DATABASE_URL="file:npci_upi.db?_pragma=busy_timeout(5000)" ./bin/server
# Server on :8080

# Test
curl http://localhost:8080/health
```

---

## Continuous Improvement

### Future optimizations

1. **Connection pooling**: Tune `sql.DB` pool size (currently: 10 open, 5 idle).
2. **Prepared statements**: Cache compiled queries for high-frequency operations.
3. **Batch posting**: Group ledger inserts for bulk reconciliation runs.
4. **Partitioned outbox**: Split `outbox_events` by tenant/account for parallel relay.

### Migration path to PostgreSQL (if needed)

- Schema is portable SQL (no Go-specific DDL).
- Replace `modernc.org/sqlite` with `github.com/lib/pq` in `go.mod`.
- Update connection string only.
- No business logic changes required.

---

## Security considerations

1. **SQL injection**: Parameterized queries throughout (no string interpolation).
2. **Decimal overflow**: Using `shopspring/decimal` (arbitrary precision).
3. **Idempotency replay attacks**: Request hash + scope key prevents duplicates.
4. **Ledger immutability**: Database-level schema prevents mutation (no UPDATE on ledger rows).

---

## Testing strategy

- **Unit tests**: `*_test.go` files in each package (chi handler tests, payment service tests).
- **Integration tests**: Docker Compose + Go test harness (phase 2).
- **Load tests**: `go-benchmark` or `vegeta` for throughput validation.

