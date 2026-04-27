# Design Doc 04 — Runtime, Reliability, and Operations

## 1. Runtime topology

1. FastAPI service replicas (stateless)
2. Orchestrator worker replicas
3. Outbox relay worker
4. Reconciliation scheduler/worker
5. PostgreSQL primary
6. Redis cache
7. Kafka broker cluster

## 2. Reliability patterns

## 2.1 Idempotency

- API idempotency via Redis fast check + persisted canonical response in DB.
- Consumer idempotency via event id tracking.

## 2.2 Outbox consistency

- Transaction + outbox insert in one DB transaction.
- Relay retries until published; publish status recorded.

## 2.3 Saga compensation

- Debit and credit are separate steps.
- If credit fails after debit, trigger deterministic reversal saga.

## 2.4 Retry and DLQ

- Exponential backoff with jitter for retriable failures.
- Non-retriable or poison messages routed to DLQ.

## 3. Operational controls

1. Manual reconciliation run endpoint.
2. Incident listing for mismatch and stuck transaction cases.
3. Admin-only override paths with audit reason.

## 4. Observability requirements

## 4.1 Metrics

- API latency p50/p95/p99
- completion latency
- queue lag
- reversal ratio
- reconciliation mismatch count

## 4.2 Logs

Structured JSON logs with:
- `service`
- `transaction_id`
- `correlation_id`
- `event_id`
- `state`
- `reason_code`

## 4.3 Tracing

- End-to-end trace from API request through worker and adapter calls.

## 5. SLO/SLA targets (simulator)

1. p95 payment initiate latency < 150 ms (normal load)
2. happy-path completion < 2 s target
3. alert if non-terminal state age exceeds configured threshold

## 6. Failure scenarios and expected behavior

1. **Duplicate request** -> same response, no duplicate postings.
2. **Worker crash after event read** -> safe replay, no duplicate effect.
3. **Debit posted, credit timeout** -> reversal flow required.
4. **Outbox relay outage** -> backlog accumulates; publish resumes without data loss.

## 7. Deployment environments

1. **Local**: Docker Compose single-node dependencies.
2. **Staging**: production-like topology with synthetic traffic.
3. **Production-sim**: HA settings, autoscaled workers, alerting enabled.

