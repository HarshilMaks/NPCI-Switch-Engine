# NPCI UPI Simulator (Backend)

UPI-inspired payment rail simulator focused on backend correctness:

- double-entry ledger
- transaction state machine
- idempotent payment creation
- automatic and manual reversal flows
- reconciliation runs

This is a simulator/sandbox project, not a real payment network integration.

## Run

```bash
python -m venv .venv
source .venv/bin/activate
pip install -e .
uvicorn app.main:app --reload --app-dir src
```

## Seeded VPAs

- `alice@bank` (active, funded)
- `bob@bank` (active, funded)
- `inactive@bank` (inactive; useful for credit-failure/reversal simulation)

## Quick flow

1. Create payment:

```bash
curl -s -X POST "http://127.0.0.1:8000/api/v1/payments" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-001" \
  -d '{"payer_vpa":"alice@bank","payee_vpa":"bob@bank","amount":"100.00","currency":"INR"}'
```

2. Get payment status:

```bash
curl -s "http://127.0.0.1:8000/api/v1/payments/<transaction_id>"
```

3. Run reconciliation:

```bash
curl -s -X POST "http://127.0.0.1:8000/api/v1/reconciliation/run"
```
