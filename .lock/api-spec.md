# API Contract Spec (Canonical Baseline)

## 1. Common headers
1. `Idempotency-Key` (required for mutating endpoints)
2. `Correlation-Id` (optional inbound, required outbound)
3. `Content-Type: application/json`

## 2. Common error schema
```json
{
  "code": "STRING_CODE",
  "message": "Human readable message",
  "details": {},
  "correlation_id": "uuid-or-trace-id"
}
```

## 3. Endpoints

## 3.1 POST /api/v1/payments
Creates a payment request asynchronously.

Request:
```json
{
  "payer_vpa": "alice@bank",
  "payee_vpa": "merchant@bank",
  "amount": "100.00",
  "currency": "INR",
  "client_ref": "optional-client-ref"
}
```

Response (202):
```json
{
  "transaction_id": "uuid",
  "status": "INITIATED",
  "accepted_at": "ISO-8601"
}
```

## 3.2 GET /api/v1/payments/{transaction_id}
Response (200):
```json
{
  "transaction_id": "uuid",
  "status": "COMPLETED",
  "amount": "100.00",
  "currency": "INR",
  "events": []
}
```

## 3.3 POST /api/v1/payments/{transaction_id}/confirm
Confirms auth step in simulator flow.

## 3.4 POST /api/v1/payments/{transaction_id}/cancel
Attempts cancellation where state allows.

## 3.5 POST /api/v1/reversals
Request:
```json
{
  "original_transaction_id": "uuid",
  "reason": "string"
}
```

## 3.6 POST /api/v1/reconciliation/run
Triggers reconciliation run.

## 3.7 GET /api/v1/accounts/{account_id}/ledger
Paginated ledger view.

## 4. Status codes baseline
1. `200` success read/action complete
2. `202` accepted for async processing
3. `400` validation error
4. `401/403` auth/authz failures
5. `404` not found
6. `409` idempotency or state conflict
7. `429` rate-limit/risk throttle
8. `500` internal error

## 5. Versioning policy
1. Path versioning (`/api/v1/...`).
2. Backward-compatible additive changes within v1.
3. Breaking changes require v2 endpoints.

