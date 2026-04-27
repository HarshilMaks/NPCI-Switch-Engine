# Security and Threat Model

## 1. Trust boundaries
1. Client -> API boundary
2. Service -> service internal boundary
3. Service -> data stores boundary
4. Async event bus boundary
5. Admin operations boundary

## 2. Key threats and controls

## 2.1 Duplicate/replay abuse
- **Threat**: repeated payment submissions.
- **Control**: mandatory idempotency keys + request hash validation.

## 2.2 Unauthorized mutation
- **Threat**: unauthorized API calls.
- **Control**: authn/authz enforcement, RBAC for admin operations.

## 2.3 Tampering with event/data consistency
- **Threat**: event dropped after DB commit.
- **Control**: transactional outbox pattern.

## 2.4 Sensitive data leakage
- **Threat**: PII/account data in logs.
- **Control**: log masking/tokenization + least-privilege access.

## 2.5 Message poisoning
- **Threat**: malformed or malicious events.
- **Control**: schema validation + DLQ isolation.

## 2.6 Secrets exposure
- **Threat**: credentials in source.
- **Control**: external secret manager/environment injection.

## 3. Security requirements baseline
1. TLS for all network traffic.
2. Service identity validation for internal calls.
3. Immutable audit records for privileged actions.
4. Dependency and image vulnerability scanning before release.

## 4. Incident response minimum
1. Identify scope (transactions/services affected).
2. Preserve logs/traces and relevant event payload hashes.
3. Apply containment and recovery steps.
4. Publish root-cause and corrective actions.

