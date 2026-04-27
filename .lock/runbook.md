# Operations Runbook

## 1. Stuck transaction (non-terminal too long)
1. Locate transaction by ID and correlation ID.
2. Inspect latest event and worker logs.
3. Check queue lag and consumer health.
4. If debit posted and credit unresolved, ensure reversal workflow is active.
5. Record incident and final resolution reason.

## 2. Reconciliation mismatch spike
1. Verify recent reconciliation run summary.
2. Group mismatches by `diff_type`.
3. Check recent deploy/config changes.
4. Re-run reconciliation on affected window.
5. Open incident for unresolved mismatches.

## 3. Outbox backlog growth
1. Check outbox relay worker health.
2. Inspect publish errors and retry logs.
3. Scale relay workers if needed.
4. Confirm `published_at` progression recovers.

## 4. High reversal ratio
1. Identify dominant failure reason codes.
2. Validate downstream adapter availability.
3. Review retry policy saturation.
4. Escalate if ratio exceeds configured threshold.

## 5. Severity model (minimum)
1. **SEV-1**: invariant breach or potential financial inconsistency.
2. **SEV-2**: sustained transaction completion degradation.
3. **SEV-3**: non-critical subsystem impairment.

