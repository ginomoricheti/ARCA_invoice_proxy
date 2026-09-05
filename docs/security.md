# Security Documentation

## Threat Model

### Assets to Protect
1. **API Keys** - Credentials for API access
2. **ARCA Credentials** - Certificates and private keys for AFIP communication
3. **Invoice Data** - Customer and transaction information
4. **Idempotency Keys** - Request deduplication tokens

### Threat Actors
- External attackers (unauthenticated)
- Compromised API keys
- Malicious authenticated users
- Insider threats

## Authentication & Authorization

### API Key Authentication

**Format:** `Authorization: Bearer sk_live_xxxxx` or `sk_test_xxxxx`

**Key Generation:**
- 32 random bytes (256 bits) encoded as hex
- Prefixed with `sk_live_` or `sk_test_`
- Total entropy: 256 bits

**Storage:**
- Only HMAC-SHA256 hash stored in database
- Pepper (secret) added before hashing: `HMAC-SHA256(pepper, raw_key)`
- Pepper stored in environment variable `API_KEY_PEPPER`
- Last 4 characters stored in plaintext for identification

**Validation:**
```go
expectedHash := HMAC-SHA256(pepper, providedKey)
constantTimeCompare(storedHash, expectedHash)
```

**Revocation:**
- Soft delete via `revoked_at` timestamp
- Immediate effect on next validation
- Revoked keys cannot be restored

**Expiration:**
- Optional `expires_at` timestamp
- Checked on every validation
- Expired keys cannot be reactivated

### Scopes (Future)
Current MVP accepts scopes but doesn't enforce them. Planned scopes:
- `invoices:write` - Issue invoices
- `invoices:read` - List/view invoices
- `apikeys:write` - Manage API keys
- `credentials:write` - Manage ARCA credentials

## Data Protection

### ARCA Credentials
- Certificate (PEM) and Private Key (PEM) stored in database
- **MVP**: Stored in plaintext in `BYTEA` columns
- **Production Requirement**: Encrypt at rest using:
  - Application-level encryption (libsodium/age)
  - Or database encryption (pgcrypto, TDE)
  - Or external secret manager (HashiCorp Vault, AWS Secrets Manager)

### Database Security
- Parameterized queries only (pgx prevents SQL injection)
- Least privilege database user
- TLS for database connections (enforce `sslmode=require` in production)
- Connection pooling with limits

### In Transit
- HTTPS enforced in production (reverse proxy termination)
- HSTS headers
- Secure cookies (if sessions added)

## Input Validation

### Request Validation
- DTO layer validates all inputs
- CUIT: 11 digits + checksum validation
- Invoice Type: Enum (A, B, C)
- Amounts: Non-negative integers (cents)
- Strings: Length limits, character restrictions

### Size Limits
Configure at reverse proxy:
- `client_max_body_size 1M` (nginx)
- Prevents DoS via large payloads

## Idempotency Security

### Key Generation
- Client-generated UUID v4 recommended
- Server accepts any string ≤64 chars
- Scoped per user (API Key)

### Collision Resistance
- Request body hashed with SHA-256
- Same key + different payload = rejected
- Prevents replay attacks with modified payloads

### State Machine
```
processing → succeeded
processing → failed
```
- No transitions from succeeded/failed
- TTL enforcement (24h default)
- Expired keys treated as not found

## Error Handling

### Information Disclosure Prevention
- No stack traces in responses
- No SQL errors exposed
- No internal file paths
- Generic messages for 5xx errors
- Structured error codes for client handling

### Logging
- Request ID correlation
- No sensitive data in logs (API keys, ARCA credentials, payloads)
- Structured JSON logging
- Audit trail for sensitive operations (key creation, revocation)

## Secrets Management

### Environment Variables
| Variable | Purpose | Required | Secret |
|----------|---------|----------|--------|
| `DATABASE_URL` | PostgreSQL connection | Yes | Yes |
| `API_KEY_PEPPER` | HMAC pepper for API keys | Yes | Yes |
| `ARCA_CERT_PATH` | Certificate file path | No | Yes |
| `ARCA_KEY_PATH` | Private key file path | No | Yes |

### .env.example
Template provided without real values. Never commit `.env` files.

### Production Recommendations
- Use secret manager (Vault, AWS Secrets Manager, GCP Secret Manager)
- Inject secrets at runtime (not build time)
- Rotate `API_KEY_PEPPER` periodically (requires re-hashing all keys)
- Rotate database passwords regularly

## ARCA Integration Security (Phase 2)

### WSAA Authentication
- CMS signed with private key
- Short-lived tokens (12h typical)
- Automatic renewal before expiry
- Private key never leaves secure storage

### WSFEv1 Communication
- SOAP over HTTPS
- Certificate validation
- Request/response logging (without sensitive data)
- Timeout and retry policies

### Certificate Management
- Expiration monitoring (alert 30 days before)
- Automated renewal workflow
- Separate certs for homologation vs production

## Network Security

### Recommended Production Setup
```
Internet → WAF/CDN → Load Balancer → API (private subnet) → Database (private subnet)
                              ↘ Secret Manager
```

### Firewall Rules
- API only accepts from load balancer
- Database only accepts from API
- No direct internet access for API or DB

### TLS Configuration
- TLS 1.2 minimum
- Strong cipher suites
- Certificate pinning for ARCA endpoints (if possible)

## Compliance

### Data Retention
- Invoices: Per Argentine tax law (10 years)
- API Keys: Until revoked + 90 days
- Idempotency Keys: 24 hours (configurable)
- Logs: 90 days (structured), 1 year (audit)

### GDPR / Argentine Law 25.326
- Personal data: CUIT, email, name, address
- Right to deletion: Soft delete with anonymization
- Data processing agreement with subprocessors

## Security Checklist for Production

- [ ] Change default `API_KEY_PEPPER`
- [ ] Enable database TLS (`sslmode=verify-full`)
- [ ] Configure reverse proxy with rate limiting
- [ ] Set up WAF rules
- [ ] Enable structured audit logging
- [ ] Configure secret manager for ARCA credentials
- [ ] Set up certificate expiration monitoring
- [ ] Configure backup encryption
- [ ] Run security scanning (gosec, trivy)
- [ ] Penetration testing
- [ ] Incident response plan
- [ ] Regular dependency updates (Dependabot/Renovate)

## Incident Response

### API Key Compromise
1. Revoke compromised key immediately
2. Audit recent usage (check logs)
3. Issue replacement key
4. Notify customer

### ARCA Credential Compromise
1. Revoke certificate at AFIP
2. Generate new key pair
3. Update in secret manager
4. Rotate `API_KEY_PEPPER` if database accessed

### Database Breach
1. Rotate database credentials
2. Rotate `API_KEY_PEPPER` (invalidate all API keys)
3. Notify affected customers
4. Audit access logs