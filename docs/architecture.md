# Architecture Documentation

## Overview

ARCA Invoice Proxy is a B2B SaaS backend that abstracts the complexity of electronic invoicing with ARCA (ex AFIP) in Argentina. The system provides a clean REST API for developers and no-code tools to issue electronic invoices without dealing with WSAA authentication, CMS signing, or WSFEv1 SOAP services directly.

## Architectural Style

The project follows a **modular monolith** architecture with clear separation of concerns using the **Clean Architecture** pattern:

```
cmd/api/                 # Application entry point
internal/
  domain/                # Enterprise business rules (no external dependencies)
    invoice/             # Invoice domain model
    customer/            # Customer domain model
    credential/          # ARCA credentials domain model
    apikey/              # API Key domain model
    idempotency/         # Idempotency domain model
    errors/              # Error definitions
  application/           # Application business rules (use cases)
    invoice/             # IssueInvoice use case
    authentication/      # API Key authentication
    idempotency/         # Idempotency processing
  infrastructure/        # External adapters
    postgres/            # PostgreSQL repositories & connection
  interfaces/            # Interface adapters
    http/                # HTTP handlers, middleware, DTOs
migrations/              # SQL database migrations
docs/                    # Documentation
tests/                   # Test files
```

## Layer Responsibilities

### Domain Layer (`internal/domain/`)
- Contains pure business logic and rules
- No dependencies on external frameworks, databases, or HTTP
- Defines entities, value objects, and domain interfaces
- Uses strong typing (custom types for InvoiceType, IVACondition, etc.)

### Application Layer (`internal/application/`)
- Orchestrates domain objects to fulfill use cases
- Depends only on domain layer and interfaces (not implementations)
- Contains use cases: `IssueInvoice`, `AuthenticateAPIKey`, `ProcessIdempotency`
- Defines repository interfaces that infrastructure implements

### Infrastructure Layer (`internal/infrastructure/`)
- Implements repository interfaces defined in application layer
- Handles PostgreSQL connections, queries, transactions
- Provides concrete ARCA client implementation (mock in MVP)

### Interfaces Layer (`internal/interfaces/`)
- Adapts external requests to internal use cases
- HTTP handlers, middleware, request/response DTOs
- Input validation, serialization, error mapping

## Key Design Decisions

### 1. Dependency Rule
Dependencies point inward: `interfaces → application → domain ← infrastructure`
- Domain has zero external dependencies
- Application depends only on domain interfaces
- Infrastructure implements application interfaces
- Interfaces depend on application

### 2. Interface Segregation
Small, focused interfaces:
- `InvoiceRepository` - only invoice operations
- `UserRepository` - only user operations
- `ARCAClient` - only ARCA operations
- `IdempotencyStore` - only idempotency operations

### 3. Strong Typing
- `InvoiceType` instead of string
- `IVACondition` instead of string
- `Environment` instead of string
- `IdempotencyStatus` instead of string
- Custom types with validation constructors

### 4. Error Handling
- Custom `AppError` with codes, messages, and details
- Error codes map to HTTP status codes
- No stack traces or internal details in responses
- Errors are wrapped with context using `errors.Wrap`

### 5. Idempotency
- Implemented at application layer
- Uses request hash (SHA-256) to detect payload changes
- Stores state in PostgreSQL with TTL
- States: `processing`, `succeeded`, `failed`

### 6. API Key Authentication
- Keys hashed with HMAC-SHA256 using pepper
- Never stored in plain text
- Prefix identifies environment (`sk_live_` vs `sk_test_`)
- Supports revocation and expiration

### 7. ARCA Abstraction
- `ARCAClient` interface in domain
- Mock implementation for MVP
- Real implementation (WSAA/WSFEv1) planned for phase 2
- Credential management separated from ARCA communication

## Data Flow: Issue Invoice

```
HTTP Request (POST /v1/facturar)
    ↓
RequestID Middleware (adds correlation ID)
    ↓
Authentication Middleware (validates Bearer token)
    ↓
InvoiceHandler (parses/validates DTO)
    ↓
Idempotency Service (checks/creates idempotency record)
    ↓
IssueInvoice Use Case
    ├─ Validate input
    ├─ Get user & ARCA credentials
    ├─ Create Invoice domain object
    ├─ Call ARCA Client (GetLastVoucher → CreateVoucher)
    ├─ Handle ARCA response (issued/rejected)
    ├─ Persist Invoice + Items
    └─ Complete idempotency record
    ↓
HTTP Response (201 Created + Invoice JSON)
```

## Concurrency Considerations

- Idempotency keys use `ON CONFLICT` upsert for atomic creation
- Invoice creation uses database transactions
- Database connection pooling configured via pgxpool
- Context propagation for cancellation/timeout

## Security Considerations

- API keys hashed with pepper (not just salt)
- No secrets in logs or error responses
- Parameterized SQL queries (no injection)
- Input validation at DTO layer
- Request size limits (to be configured in reverse proxy)
- CORS and security headers (to be configured in reverse proxy)

## Future Extensibility

- ARCA client implementation swap (interface-based)
- Multiple ARCA environments per customer
- Webhook notifications for invoice status changes
- Rate limiting middleware
- Audit logging
- Multi-tenancy support (already single-tenant per API key)