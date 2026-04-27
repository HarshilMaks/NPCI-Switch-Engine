# Design Doc 03 — API and Event Contract Design

## 1. API endpoints (v1)

1. `POST /api/v1/payments`
2. `GET /api/v1/payments/{transaction_id}`
3. `POST /api/v1/payments/{transaction_id}/confirm`
4. `POST /api/v1/payments/{transaction_id}/cancel`
5. `POST /api/v1/reversals`
6. `POST /api/v1/reconciliation/run`
7. `GET /api/v1/accounts/{account_id}/ledger`

## 2. API contract standards

1. `Idempotency-Key` required for all write endpoints.
2. `Correlation-Id` accepted/generated and returned in response headers.
3. Error envelope:
   - `code`
   - `message`
   - `details`
   - `correlation_id`
4. Successful mutation may return `202 Accepted` for async completion.

## 3. Request/response shape (summary)

## 3.1 POST /payments

Request fields:
- `payer_vpa`
- `payee_vpa`
- `amount`
- `currency`
- `client_ref`

Response fields:
- `transaction_id`
- `status`
- `accepted_at`

## 3.2 GET /payments/{id}

Response fields:
- `transaction_id`
- `status`
- `amount`
- `currency`
- `timeline` (events summary)

## 4. Event topics

1. `payment.initiated`
2. `payment.authorized`
3. `payment.debit_posted`
4. `payment.credit_posted`
5. `payment.completed`
6. `payment.failed`
7. `payment.reversal_pending`
8. `payment.reversed`
9. `reconciliation.diff_detected`

## 5. Event envelope standard

- `event_id`
- `event_type`
- `aggregate_type`
- `aggregate_id`
- `occurred_at`
- `schema_version`
- `correlation_id`
- `payload`

## 6. Compatibility policy

1. Additive changes only in same major version.
2. Breaking changes require `/v2`.
3. Event consumers must ignore unknown fields.

