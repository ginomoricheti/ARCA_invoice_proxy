# Database Documentation

## Overview

PostgreSQL 16+ database schema for ARCA Invoice Proxy.

## Entity Relationship Diagram

```
┌─────────────┐       ┌─────────────┐       ┌──────────────────┐
│    users    │       │  api_keys   │       │ arca_credentials │
├─────────────┤       ├─────────────┤       ├──────────────────┤
│ id (PK)     │◄──────│ user_id (FK)│       │ user_id (FK)     │
│ cuit (UK)   │       │ id (PK)     │       │ id (PK)          │
│ name        │       │ prefix      │       │ environment      │
│ email       │       │ hash        │       │ cuit             │
│ address     │       │ last_four   │       │ cert_pem         │
│ iva_cond    │       │ scopes[]    │       │ key_pem          │
│ country     │       │ expires_at  │       │ expires_at       │
│ created_at  │       │ revoked_at  │       │ created_at       │
│ updated_at  │       │ created_at  │       │ updated_at       │
└─────────────┘       │ updated_at  │       └──────────────────┘
                      └─────────────┘
                             │
                             ▼
                    ┌─────────────────┐       ┌──────────────────┐
                    │   invoices      │       │ invoice_items    │
                    ├─────────────────┤       ├──────────────────┤
                    │ id (PK)         │◄──────│ invoice_id (FK)  │
                    │ user_id (FK)    │       │ id (PK)          │
                    │ invoice_type    │       │ description      │
                    │ point_of_sale   │       │ quantity         │
                    │ status          │       │ unit_price_cents │
                    │ subtotal_cents  │       │ total_cents      │
                    │ tax_cents       │       │ created_at       │
                    │ total_cents     │       └──────────────────┘
                    │ cae             │
                    │ cae_expires_at  │
                    │ voucher_number  │
                    │ voucher_type    │
                    │ arca_result     │
                    │ arca_observations│
                    │ idempotency_key │
                    │ issued_at       │
                    │ created_at      │
                    │ updated_at      │
                    └─────────────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │ idempotency_keys │
                    ├──────────────────┤
                    │ key (PK)         │
                    │ user_id (FK)     │
                    │ request_hash     │
                    │ status           │
                    │ response_body    │
                    │ error_message    │
                    │ created_at       │
                    │ updated_at       │
                    │ expires_at       │
                    └──────────────────┘
```

## Tables

### users
Stores SaaS customers (issuers of invoices).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(36) | PK | UUID |
| cuit | VARCHAR(11) | NOT NULL, UNIQUE | 11-digit CUIT |
| name | VARCHAR(255) | NOT NULL | Business name |
| email | VARCHAR(255) | | Contact email |
| address | TEXT | NOT NULL | Fiscal address |
| iva_condition | VARCHAR(2) | NOT NULL | RI, MT, EX, CF, NC |
| country_code | CHAR(2) | NOT NULL, DEFAULT 'AR' | ISO country code |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Auto-updated via trigger |

**Indexes:**
- `idx_users_cuit` (UNIQUE) - Fast CUIT lookups
- `idx_users_email` - Email lookups

### api_keys
Stores hashed API keys for authentication.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(36) | PK | UUID |
| user_id | VARCHAR(36) | FK → users(id), CASCADE | Owner |
| name | VARCHAR(255) | NOT NULL | Human-readable name |
| prefix | VARCHAR(20) | NOT NULL | `sk_live_` or `sk_test_` |
| hash | VARCHAR(64) | NOT NULL | HMAC-SHA256(pepper, key) |
| last_four | VARCHAR(4) | NOT NULL | Last 4 chars for display |
| scopes | TEXT[] | NOT NULL, DEFAULT '{}' | Permission scopes |
| expires_at | TIMESTAMPTZ | | Optional expiration |
| revoked_at | TIMESTAMPTZ | | Soft delete timestamp |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Auto-updated |

**Indexes:**
- `idx_api_keys_user_id` - List keys by user
- `idx_api_keys_prefix_hash` - Auth lookup (prefix + last_four → hash)
- `idx_api_keys_last_four` - Display/lookup by last four

### arca_credentials
Stores ARCA certificates and private keys per user per environment.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(36) | PK | UUID |
| user_id | VARCHAR(36) | FK → users(id), CASCADE | Owner |
| environment | VARCHAR(20) | NOT NULL, CHECK | `homologation` or `production` |
| cuit | VARCHAR(11) | NOT NULL | CUIT for this credential |
| cert_pem | BYTEA | NOT NULL | X.509 certificate PEM |
| key_pem | BYTEA | NOT NULL | Private key PEM |
| expires_at | TIMESTAMPTZ | NOT NULL | Certificate expiration |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Auto-updated |

**Indexes:**
- `idx_arca_credentials_user_id` - List by user
- `idx_arca_credentials_cuit_env` - Unique per CUIT+environment

### invoices
Stores issued invoices with ARCA response data.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(36) | PK | UUID |
| user_id | VARCHAR(36) | FK → users(id), CASCADE | Issuer |
| invoice_type | CHAR(1) | NOT NULL, CHECK | A, B, or C |
| point_of_sale | INTEGER | NOT NULL, CHECK >0 | POS number |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'pending', CHECK | pending, issued, rejected, cancelled |
| subtotal_cents | BIGINT | NOT NULL, DEFAULT 0 | Subtotal in cents |
| tax_cents | BIGINT | NOT NULL, DEFAULT 0 | Tax in cents |
| total_cents | BIGINT | NOT NULL, DEFAULT 0 | Total in cents |
| cae | VARCHAR(14) | | ARCA Authorization Code |
| cae_expires_at | DATE | | CAE expiration |
| voucher_number | BIGINT | | ARCA voucher number |
| voucher_type | INTEGER | | ARCA voucher type code |
| arca_result | VARCHAR(20) | | ARCA result code |
| arca_observations | JSONB | | ARCA observations array |
| idempotency_key | VARCHAR(64) | | Client-provided idempotency key |
| issued_at | TIMESTAMPTZ | | When ARCA accepted |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Auto-updated |

**Indexes:**
- `idx_invoices_user_id` - List by user
- `idx_invoices_status` - Filter by status
- `idx_invoices_idempotency_key` - Idempotency lookup
- `idx_invoices_created_at` - Time-range queries
- `idx_invoices_user_idempotency` (UNIQUE, PARTIAL) - Prevent duplicate idempotency per user

### invoice_items
Line items for each invoice.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(36) | PK | UUID |
| invoice_id | VARCHAR(36) | FK → invoices(id), CASCADE | Parent invoice |
| description | TEXT | NOT NULL | Item description |
| quantity | INTEGER | NOT NULL, CHECK >0 | Quantity |
| unit_price_cents | BIGINT | NOT NULL, CHECK >=0 | Unit price in cents |
| total_cents | BIGINT | NOT NULL, CHECK >=0 | Line total (qty × price) |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

**Indexes:**
- `idx_invoice_items_invoice_id` - Items by invoice

### idempotency_keys
Stores idempotency state for request deduplication.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| key | VARCHAR(64) | PK | Client-provided key |
| user_id | VARCHAR(36) | FK → users(id), CASCADE | Owner (scoped) |
| request_hash | VARCHAR(64) | NOT NULL | SHA-256 of request body |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'processing', CHECK | processing, succeeded, failed |
| response_body | BYTEA | | Cached success response |
| error_message | TEXT | | Error message if failed |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Auto-updated |
| expires_at | TIMESTAMPTZ | NOT NULL | TTL expiration |

**Indexes:**
- `idx_idempotency_keys_user_id` - Cleanup by user
- `idx_idempotency_keys_expires_at` - TTL cleanup job
- `idx_idempotency_keys_status` - Filter by status

## Triggers

### `update_updated_at_column()`
Auto-updates `updated_at` timestamp on row modification.

Applied to: `users`, `api_keys`, `arca_credentials`, `invoices`, `idempotency_keys`

## Migrations

Located in `migrations/` directory:
- `001_initial_schema.sql` - Complete initial schema

Run with:
```bash
# Using psql
psql -d arca_proxy -f migrations/001_initial_schema.sql

# Or with golang-migrate
migrate -path migrations -database "postgres://..." up
```

## Common Queries

### Get user with API keys
```sql
SELECT u.*, json_agg(ak.*) as api_keys
FROM users u
LEFT JOIN api_keys ak ON ak.user_id = u.id AND ak.revoked_at IS NULL
WHERE u.cuit = '20123456789'
GROUP BY u.id;
```

### Get invoice with items
```sql
SELECT i.*, json_agg(ii.*) as items
FROM invoices i
LEFT JOIN invoice_items ii ON ii.invoice_id = i.id
WHERE i.id = 'inv_...'
GROUP BY i.id;
```

### Cleanup expired idempotency keys
```sql
DELETE FROM idempotency_keys WHERE expires_at < NOW();
```

### Find invoices by date range
```sql
SELECT * FROM invoices
WHERE user_id = 'cust_123'
  AND created_at BETWEEN '2026-01-01' AND '2026-01-31'
ORDER BY created_at DESC;
```

## Data Types Reference

| Domain Type | PostgreSQL Type | Notes |
|-------------|-----------------|-------|
| ID (UUID) | VARCHAR(36) | Generated by application |
| CUIT | VARCHAR(11) | 11 digits, validated in app |
| Money | BIGINT (cents) | Avoids floating point |
| Timestamps | TIMESTAMPTZ | Always UTC |
| Enum | VARCHAR + CHECK | Enforced at DB level |
| JSON | JSONB | ARCA observations |
| Binary | BYTEA | Certificates, keys, cached responses |

## Connection Pooling

Configured via `pgxpool`:
- `MaxConns`: 25 (configurable)
- `MinConns`: 5 (configurable)
- `MaxConnLifetime`: 5 minutes
- `MaxConnIdleTime`: 5 minutes
- `HealthCheckPeriod`: 1 minute

## Backup Strategy

- Daily full backup (pg_dump)
- WAL archiving for point-in-time recovery
- Test restore monthly
- Encrypt backups at rest

## Monitoring

Key metrics to monitor:
- Connection pool usage
- Query latency (p50, p95, p99)
- Idempotency key cleanup lag
- Invoice status distribution
- ARCA credential expiration (alert 30 days before)