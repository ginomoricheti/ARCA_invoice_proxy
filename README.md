# ARCA Invoice Proxy

> **Status: MVP Implementation Complete - Pending Validation in Executable Environment**

A Go backend API that abstracts the complexity of electronic invoicing with ARCA (ex AFIP) in Argentina.

## Purpose

ARCA (Administraci+¦n Federal de Ingresos P+¦blicos, formerly AFIP) requires complex integration for electronic invoicing:
- WSAA authentication with CMS signing
- WSFEv1 SOAP web service
- Certificate management
- CAE (C+¦digo de Autorizaci+¦n Electr+¦nica) handling

This API provides a simple REST interface:
```http
POST /v1/facturar
Authorization: Bearer sk_live_xxxxx
Idempotency-Key: uuid-v4

{
  "cuit": "20123456789",
  "tipo_factura": "C",
  "punto_venta": 1,
  "items": [
    {"descripcion": "Servicio de desarrollo", "cantidad": 1, "precio_unitario": 1000000}
  ]
}
```

## Architecture

```
GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ
Göé                      Clean Architecture                      Göé
Gö£GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGö¼GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGö¼GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöñ
Göé     Domain       Göé  Application     Göé    Infrastructure      Göé
Göé  (Pure Go)       Göé  (Use Cases)     Göé   (PostgreSQL, HTTP)   Göé
Gö£GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGö+GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGö+GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöñ
Göé GÇó Invoice        Göé GÇó IssueInvoice   Göé GÇó pgxpool              Göé
Göé GÇó Customer       Göé GÇó Authenticate   Göé GÇó Repositories         Göé
Göé GÇó Credential     Göé GÇó Idempotency    Göé GÇó Mock ARCA Client     Göé
Göé GÇó APIKey         Göé                  Göé                        Göé
Göé GÇó Idempotency    Göé                  Göé                        Göé
Göé GÇó Errors         Göé                  Göé                        Göé
GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGö¦GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGö¦GöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ
                              Göé
                              Gû+
                    GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ
                    Göé   Interfaces     Göé
                    Göé   (HTTP Layer)   Göé
                    Göé GÇó Handlers       Göé
                    Göé GÇó Middleware     Göé
                    Göé GÇó DTOs           Göé
                    GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ
```

**Key Principles:**
- Domain has zero external dependencies
- Dependencies point inward (Clean Architecture)
- Small interfaces, explicit errors
- Standard library preferred over frameworks

## Features (MVP)

| Feature | Status |
|---------|--------|
| POST /v1/facturar | G£à Implemented |
| API Key Authentication | G£à Implemented |
| Idempotency (Idempotency-Key) | G£à Implemented |
| PostgreSQL Persistence | G£à Implemented |
| Invoice Types A, B, C | G£à Implemented |
| CUIT Validation (checksum) | G£à Implemented |
| ARCA Integration | =ƒöä Mock (Phase 2) |
| OpenAPI Spec | G£à Documented |
| Docker Support | G£à Configured |

## Quick Start

### Prerequisites
- Go 1.23+ (for building)
- PostgreSQL 16+
- Docker & Docker Compose (recommended)

### Configuration

Copy `.env.example` to `.env` and adjust:
```bash
cp .env.example .env
# Edit .env with your values
```

Key variables:
```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/arca_proxy?sslmode=disable
API_KEY_PEPPER=your-secure-random-pepper-here
ARCA_ENVIRONMENT=homologation
```

### Running with Docker Compose

```bash
docker-compose up -d
```

This starts:
- PostgreSQL on port 5432
- API on port 8080

### Running Locally

```bash
# Start PostgreSQL (or use docker-compose up -d postgres)
# Run migrations
psql -d arca_proxy -f migrations/001_initial_schema.sql

# Build and run
go build -o arca-invoice-proxy ./cmd/api
./arca-invoice-proxy
```

### API Usage

**1. Create an API Key** (requires existing user in DB - see note below)
```bash
curl -X POST http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Integration", "prefix": "sk_test_", "scopes": ["invoices:write"]}'
```

**2. Issue an Invoice**
```bash
curl -X POST http://localhost:8080/v1/facturar \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "cuit": "20123456789",
    "tipo_factura": "C",
    "punto_venta": 1,
    "items": [
      {"descripcion": "Servicio de desarrollo", "cantidad": 1, "precio_unitario": 1000000}
    ]
  }'
```

**3. List API Keys**
```bash
curl -X GET http://localhost:8080/v1/api-keys \
  -H "Authorization: Bearer sk_test_yourkey"
```

**4. Revoke API Key**
```bash
curl -X DELETE http://localhost:8080/v1/api-keys/key_abc123 \
  -H "Authorization: Bearer sk_test_yourkey"
```

> **Note**: The MVP requires a user to exist in the database before creating API keys. In Phase 2, a user registration/onboarding flow will be added. For now, insert a user directly:
```sql
INSERT INTO users (id, cuit, name, email, address, iva_condition, country_code)
VALUES ('cust_123', '20123456789', 'Test Company', 'test@example.com', 'Av. Corrientes 1234', 'RI', 'AR');
```

## Project Structure

```
.
Gö£GöÇGöÇ cmd/api/main.go              # Entry point
Gö£GöÇGöÇ internal/
Göé   Gö£GöÇGöÇ config/                  # Configuration
Göé   Gö£GöÇGöÇ domain/                  # Domain layer
Göé   Göé   Gö£GöÇGöÇ invoice/             # Invoice entity, types
Göé   Göé   Gö£GöÇGöÇ customer/            # Customer entity, CUIT validation
Göé   Göé   Gö£GöÇGöÇ credential/          # ARCA credentials, ARCAClient interface
Göé   Göé   Gö£GöÇGöÇ apikey/              # API Key entity, hashing
Göé   Göé   Gö£GöÇGöÇ idempotency/         # Idempotency entity, store interface
Göé   Göé   GööGöÇGöÇ errors/              # Error codes, AppError
Göé   Gö£GöÇGöÇ application/             # Application layer (use cases)
Göé   Göé   Gö£GöÇGöÇ invoice/             # IssueInvoice service
Göé   Göé   Gö£GöÇGöÇ authentication/      # API Key auth service
Göé   Göé   GööGöÇGöÇ idempotency/         # Idempotency processing
Göé   Gö£GöÇGöÇ infrastructure/
Göé   Göé   GööGöÇGöÇ postgres/            # pgxpool, repositories
Göé   GööGöÇGöÇ interfaces/http/         # HTTP layer
Göé       Gö£GöÇGöÇ handlers/            # Invoice, APIKey, Health handlers
Göé       Gö£GöÇGöÇ middleware/          # Auth, RequestID middleware
Göé       GööGöÇGöÇ dto/                 # Request/Response DTOs
Gö£GöÇGöÇ migrations/
Göé   GööGöÇGöÇ 001_initial_schema.sql   # Database schema
Gö£GöÇGöÇ docs/
Göé   Gö£GöÇGöÇ openapi.yaml             # OpenAPI 3.0 spec
Göé   Gö£GöÇGöÇ architecture.md          # Architecture decisions
Göé   Gö£GöÇGöÇ api.md                   # API documentation
Göé   Gö£GöÇGöÇ security.md              # Security documentation
Göé   GööGöÇGöÇ database.md              # Database schema docs
Gö£GöÇGöÇ tests/                       # Unit tests
Gö£GöÇGöÇ Dockerfile                   # Multi-stage build
Gö£GöÇGöÇ docker-compose.yml           # Local development stack
Gö£GöÇGöÇ .env.example                 # Environment template
Gö£GöÇGöÇ go.mod / go.sum              # Dependencies
GööGöÇGöÇ README.md                    # This file
```

## Dependencies

| Package | Purpose | Why Necessary |
|---------|---------|---------------|
| `github.com/jackc/pgx/v5` | PostgreSQL driver | Mature, fast, standard |
| `github.com/jackc/pgx/v5/pgxpool` | Connection pooling | Built-in pooling |
| `github.com/google/uuid` | UUID generation | Standard, no dependencies |
| `golang.org/x/crypto` | HMAC, crypto | Standard library extension |

## Testing

```bash
# Run all tests (when executable environment available)
go test ./...

# Run specific package
go test ./tests/...

# With coverage
go test -cover ./...
```

Test files created:
- `tests/domain_invoice_test.go` - Invoice domain logic
- `tests/domain_customer_test.go` - Customer domain logic
- `tests/domain_apikey_test.go` - API Key domain logic
- `tests/domain_idempotency_test.go` - Idempotency domain logic
- `tests/application_invoice_test.go` - IssueInvoice use case
- `tests/application_auth_test.go` - Authentication service

## Documentation

- [Architecture](docs/architecture.md) - Design decisions, layer responsibilities
- [API Reference](docs/api.md) - Endpoints, request/response formats, examples
- [Security](docs/security.md) - Threat model, authentication, data protection
- [Database](docs/database.md) - Schema, indexes, queries, ERD
- [OpenAPI Spec](docs/openapi.yaml) - Machine-readable API contract

## ARCA Integration (Phase 2)

The MVP uses a mock ARCA client. Phase 2 will implement:

```
GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ     GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ     GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ
Göé   WSAA      GöéGöÇGöÇGöÇGöÇGû¦Göé  LoginCms   GöéGöÇGöÇGöÇGöÇGû¦Göé   Token     Göé
Göé  (Auth)     Göé     Göé  (SOAP)     Göé     Göé  (12h TTL)  Göé
GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ     GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ     GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ
                           Göé
                           Gû+
GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ     GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ     GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ
Göé  WSFEv1     GöéGùäGöÇGöÇGöÇGöé FECAESolicitGöéGùäGöÇGöÇGöÇGöé  Invoice    Göé
Göé  (Invoice)  Göé     Göé    (SOAP)   Göé     Göé   Data      Göé
GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ     GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ     GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ
                           Göé
                           Gû+
                    GöîGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÉ
                    Göé    CAE      Göé
                    Göé  (Response) Göé
                    GööGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÇGöÿ
```

Components to implement:
- CMS signing (golang.org/x/crypto/pkcs7)
- WSAA client (SOAP 1.2)
- WSFEv1 client (SOAP 1.2)
- Certificate lifecycle management
- CAE/QR/PDF generation

## Security

- API Keys hashed with HMAC-SHA256 + pepper
- No secrets in logs or error responses
- Parameterized SQL queries
- Input validation at DTO layer
- Idempotency with request hash verification
- See [Security Docs](docs/security.md) for details

## License

MIT License - See LICENSE file for details.

## Contributing

1. Fork the repository
2. Create feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit pull request

## Roadmap

- [ ] Phase 2: Real ARCA Integration (WSAA + WSFEv1)
- [ ] Phase 2: User registration & onboarding
- [ ] Phase 2: Webhook notifications
- [ ] Phase 2: PDF/QR generation
- [ ] Rate limiting middleware
- [ ] Audit logging
- [ ] Multi-environment ARCA credentials per user
- [ ] Metrics & tracing (OpenTelemetry)
