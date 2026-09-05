# Domain Contracts Documentation

This document describes all domain contracts (interfaces) in the ARCA Invoice Proxy system, their purpose, methods, and implementations.

---

## Table of Contents

1. [Repository Contracts](#repository-contracts)
   - [UserRepository](#userrepository)
   - [APIKeyRepository](#apikeyrepository)
   - [ARCACredentialRepository](#arcacredentialrepository)
   - [InvoiceRepository](#invoicerepository)
   - [IdempotencyStore](#idempotencystore)
2. [External Service Contracts](#external-service-contracts)
   - [ARCAClient](#arcaclient)
3. [Application Service Contracts](#application-service-contracts)
   - [AuthenticationService](#authenticationservice)
   - [IdempotencyService](#idempotencyservice)
   - [InvoiceService](#invoiceservice)
4. [Domain Model Contracts](#domain-model-contracts)
   - [Invoice](#invoice)
   - [Customer](#customer)
   - [ARCACredential](#arcacredential)
   - [APIKey](#apikey)
   - [IdempotencyRecord](#idempotencyrecord)

---

## Repository Contracts

### UserRepository

**Location:** `internal/domain/customer/repository.go` (interface) → `internal/infrastructure/postgres/repositories.go` (implementation)

**Purpose:** Persistence abstraction for Customer (User) entities.

```go
type UserRepository interface {
    Create(ctx context.Context, c *Customer) error
    GetByCUIT(ctx context.Context, cuit string) (*Customer, error)
    GetByID(ctx context.Context, id string) (*Customer, error)
}
```

| Method | Description | Error Codes |
|--------|-------------|-------------|
| `Create` | Inserts new customer | `database_error`, `validation_failed` |
| `GetByCUIT` | Finds by CUIT (unique) | `not_found`, `database_error` |
| `GetByID` | Finds by UUID | `not_found`, `database_error` |

**Implementation Notes:**
- Uses `pgxpool` for connection pooling
- CUIT has unique constraint in DB
- `updated_at` auto-managed by DB trigger
- Returns domain `Customer` entity, not DTO

---

### APIKeyRepository

**Location:** `internal/domain/apikey/apikey.go` (interface) → `internal/infrastructure/postgres/repositories.go` (implementation)

**Purpose:** Manages API Key lifecycle (create, validate, list, revoke).

```go
type APIKeyStore interface {
    Create(ctx context.Context, key *APIKey) error
    GetByPrefixAndSuffix(ctx context.Context, prefix, suffix string) (*APIKey, error)
    GetByID(ctx context.Context, id string) (*APIKey, error)
    ListByCustomer(ctx context.Context, customerID string) ([]*APIKey, error)
    Update(ctx context.Context, key *APIKey) error
}
```

| Method | Description | Error Codes |
|--------|-------------|-------------|
| `Create` | Stores hashed key | `database_error` |
| `GetByPrefixAndSuffix` | Auth lookup (prefix + last4) | `not_found`, `database_error` |
| `GetByID` | Admin lookup by ID | `not_found`, `database_error` |
| `ListByCustomer` | Returns all keys for user | `database_error` |
| `Update` | Revoke, rename, modify scopes | `database_error` |

**Security Notes:**
- Only stores HMAC-SHA256 hash + last 4 chars
- Never stores raw key
- Prefix (`sk_live_`/`sk_test_`) used for routing
- Soft delete via `revoked_at` timestamp

---

### ARCACredentialRepository

**Location:** `internal/domain/credential/credential.go` (interface) → `internal/infrastructure/postgres/repositories.go` (implementation)

**Purpose:** Manages ARCA certificates per customer per environment.

```go
type ARCACredentialRepository interface {
    Create(ctx context.Context, c *ARCACredential) error
    GetByCustomerAndEnvironment(ctx context.Context, customerID string, env Environment) (*ARCACredential, error)
    GetByID(ctx context.Context, id string) (*ARCACredential, error)
}
```

| Method | Description | Error Codes |
|--------|-------------|-------------|
| `Create` | Stores cert + key PEM | `database_error` |
| `GetByCustomerAndEnvironment` | Runtime lookup for invoicing | `not_found`, `database_error` |
| `GetByID` | Admin lookup | `not_found`, `database_error` |

**Security Notes:**
- **MVP:** Stores PEM in plaintext `BYTEA`
- **Production:** Must encrypt at rest (Vault, pgcrypto, app-level)
- Unique constraint on `(customer_id, environment)`
- Expiration tracked for rotation alerts

---

### InvoiceRepository

**Location:** `internal/application/invoice/issue_invoice.go` (interface) → `internal/infrastructure/postgres/repositories.go` (implementation)

**Purpose:** Full invoice lifecycle with items and ARCA response.

```go
type InvoiceRepository interface {
    Create(ctx context.Context, inv *Invoice, items []InvoiceItem) error
    GetByID(ctx context.Context, id string) (*Invoice, error)
    GetByIdempotencyKey(ctx context.Context, userID, key string) (*Invoice, error)
    Update(ctx context.Context, inv *Invoice) error
}
```

| Method | Description | Error Codes |
|--------|-------------|-------------|
| `Create` | Atomic insert invoice + items (tx) | `database_error` |
| `GetByID` | Full invoice with items + ARCA data | `not_found`, `database_error` |
| `GetByIdempotencyKey` | Idempotency lookup (user-scoped) | `not_found`, `database_error` |
| `Update` | Status, CAE, observations | `database_error` |

**Implementation Notes:**
- `Create` uses transaction for invoice + items atomicity
- `GetByIdempotencyKey` uses partial unique index
- ARCA observations stored as `JSONB`
- Monetary amounts in cents (BIGINT)

---

### IdempotencyStore

**Location:** `internal/domain/idempotency/idempotency.go` (interface) → `internal/infrastructure/postgres/repositories.go` (implementation)

**Purpose:** Request deduplication with payload verification.

```go
type IdempotencyStore interface {
    Create(ctx context.Context, record *IdempotencyRecord) error
    Get(ctx context.Context, key string) (*IdempotencyRecord, error)
    Update(ctx context.Context, record *IdempotencyRecord) error
}
```

| Method | Description | Error Codes |
|--------|-------------|-------------|
| `Create` | Upsert (ON CONFLICT) | `database_error` |
| `Get` | Retrieve by key | `idempotency_not_found`, `database_error` |
| `Update` | Complete/fail record | `database_error` |

**State Machine:**
```
Create (processing) → Complete (succeeded) → cached response
                    → Fail (failed)         → cached error
```

**Conflict Detection:**
- SHA-256 hash of request body stored
- Same key + different hash = `idempotency_conflict`
- TTL enforced via `expires_at` (default 24h)

---

## External Service Contracts

### ARCAClient

**Location:** `internal/domain/credential/credential.go` (interface)

**Purpose:** Abstraction for ARCA WSAA/WSFEv1 communication. **MVP uses mock.**

```go
type ARCAClient interface {
    GetLastVoucher(ctx context.Context, cuit string, invoiceType string, pointOfSale int) (int64, error)
    CreateVoucher(ctx context.Context, request VoucherRequest) (*VoucherResponse, error)
}
```

| Method | Description | ARCA Operation |
|--------|-------------|----------------|
| `GetLastVoucher` | Next voucher number | `FECompUltimoAutorizado` |
| `CreateVoucher` | Authorize invoice | `FECAESolicitar` |

**Request/Response Types:**

```go
type VoucherRequest struct {
    CUIT            string       // Issuer CUIT
    InvoiceType     string       // "1"=A, "6"=B, "11"=C
    PointOfSale     int          // POS number
    ConceptType     int          // 1=Products, 2=Services, 3=Both
    DocType         int          // 80=CUIT, 99=Consumidor Final
    DocNumber       string       // Client CUIT or "0"
    ServiceFrom     string       // YYYYMMDD
    ServiceTo       string       // YYYYMMDD
    ExpirationDate  string       // YYYYMMDD
    Items           []VoucherItem
    CurrencyID      string       // "PES"
    CurrencyRate    float64      // 1.0
}

type VoucherItem struct {
    Description string
    Quantity    float64
    UnitPrice   float64
    Bonus       float64
    Taxes       []VoucherTax
}

type VoucherResponse struct {
    CAE               string
    CAEExpirationDate string
    VoucherNumber     int64
    VoucherType       int
    Result            string       // "A"=Approved, "R"=Rejected
    Observations      []VoucherObservation
}
```

**Phase 2 Implementation:**
- WSAA: CMS signing → `LoginCms` → token (12h cache)
- WSFEv1: SOAP 1.2 with token header
- Certificate validation, retry logic, timeouts
- Homologation vs Production endpoints

---

## Application Service Contracts

### AuthenticationService

**Location:** `internal/application/authentication/service.go`

**Purpose:** API Key validation and management.

```go
type Service struct {
    store  APIKeyStore
    pepper string
}

func NewService(store APIKeyStore, pepper string) *Service

func (s *Service) ValidateAPIKey(ctx context.Context, rawKey string) (*APIKey, error)
func (s *Service) CreateAPIKey(ctx context.Context, customerID, name, prefix string, scopes []string, expiresAt *int64) (*APIKey, string, error)
func (s *Service) ListAPIKeys(ctx context.Context, customerID string) ([]*APIKey, error)
func (s *Service) RevokeAPIKey(ctx context.Context, keyID string) error
```

| Method | Description | Returns |
|--------|-------------|---------|
| `ValidateAPIKey` | HMAC verify + revocation/expiry check | `*APIKey` or error |
| `CreateAPIKey` | Generate + hash + store | `(*APIKey, rawKey)` |
| `ListAPIKeys` | User's keys (masked) | `[]*APIKey` |
| `RevokeAPIKey` | Soft delete | error |

**Error Codes:**
- `invalid_api_key` - Format, not found, or HMAC mismatch
- `api_key_revoked` - Key revoked
- `api_key_expired` - Key expired
- `validation_failed` - Invalid input

---

### IdempotencyService

**Location:** `internal/application/idempotency/service.go`

**Purpose:** Process requests with idempotency guarantees.

```go
type Service struct {
    store IdempotencyStore
    ttl   time.Duration
}

func NewService(store IdempotencyStore, ttl time.Duration) *Service

func (s *Service) Process(ctx context.Context, key string, requestBody []byte, handler func() (interface{}, error)) (interface{}, error)
```

**Flow:**
```
1. Check existing record by key
2. If not found: create (processing), execute handler, store result
3. If found:
   - expired → treat as not found
   - processing → return idempotency_processing
   - hash mismatch → return idempotency_conflict
   - succeeded → return cached response
   - failed → return cached error
```

**Concurrency:** DB-level upsert prevents duplicate creates.

---

### InvoiceService (IssueInvoice)

**Location:** `internal/application/invoice/issue_invoice.go`

**Purpose:** Complete invoice issuance workflow.

```go
type Service struct {
    invoiceRepo       InvoiceRepository
    userRepo          UserRepository
    credentialRepo    ARCACredentialRepository
    arcaClient        ARCAClient
    environment       credential.Environment
}

func NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, environment)

func (s *Service) IssueInvoice(ctx context.Context, input IssueInvoiceInput) (*IssueInvoiceOutput, error)
```

**Input:**
```go
type IssueInvoiceInput struct {
    CustomerID     string
    InvoiceType    string       // "A", "B", "C"
    PointOfSale    int
    Items          []InvoiceItemInput
    IdempotencyKey string
}

type InvoiceItemInput struct {
    Description    string
    Quantity       int
    UnitPriceCents int64
}
```

**Output:**
```go
type IssueInvoiceOutput struct {
    Invoice *Invoice
}
```

**Workflow:**
```
1. Check idempotency (if key provided) → return existing
2. Validate invoice type
3. Load user (customer)
4. Load ARCA credentials for environment
5. Verify credentials not expired
6. Build Invoice domain object (subtotal, validation)
7. Get last voucher number from ARCA
8. Build VoucherRequest (map types, convert cents→float)
9. Call ARCA CreateVoucher
10. Handle response:
    - Success (Result="A") → MarkIssued, persist
    - Rejected → MarkRejected, persist, return error
    - Error → MarkRejected, persist, return error
11. Complete idempotency record
12. Return invoice
```

**Error Codes:**
- `invalid_invoice_type` - Not A/B/C
- `invalid_amount` - Item validation failed
- `arca_authentication_failed` - No credentials or expired
- `arca_service_unavailable` - ARCA client error
- `arca_rejected` - ARCA returned rejection
- `database_error` - Persistence failure

---

## Domain Model Contracts

### Invoice

**Location:** `internal/domain/invoice/invoice.go`

**Entity:** Aggregate root for invoicing.

```go
type Invoice struct {
    ID              string
    CustomerID      string
    Type            InvoiceType      // A, B, C
    PointOfSale     int
    Items           []InvoiceItem
    Status          InvoiceStatus    // pending, issued, rejected, cancelled
    SubtotalCents   int64
    TaxCents        int64
    TotalCents      int64
    ARCAResponse    *ARCAResponse
    IdempotencyKey  string
    CreatedAt       time.Time
    UpdatedAt       time.Time
    IssuedAt        *time.Time
}
```

**Factory:**
```go
func NewInvoice(customerID string, invoiceType InvoiceType, pointOfSale int, items []InvoiceItem, idempotencyKey string) (*Invoice, error)
```

**State Transitions:**
```
pending → issued    (MarkIssued + ARCAResponse)
pending → rejected  (MarkRejected + observations)
*       → cancelled (MarkCancelled)
```

**Validation Rules:**
- At least 1 item
- Point of sale > 0
- Valid InvoiceType
- All items valid (description, quantity>0, price>=0)

---

### InvoiceItem

**Location:** `internal/domain/invoice/invoice.go`

**Value Object:** Invoice line item.

```go
type InvoiceItem struct {
    Description    string
    Quantity       int
    UnitPriceCents int64
}

func NewInvoiceItem(description string, quantity int, unitPriceCents int64) (InvoiceItem, error)
func (i InvoiceItem) TotalCents() int64
```

**Validation:**
- Description required
- Quantity > 0
- UnitPriceCents >= 0
- Total = Quantity × UnitPriceCents

---

### InvoiceType

**Location:** `internal/domain/invoice/invoice.go`

**Enumeration:** Strong-typed invoice classification.

```go
type InvoiceType string

const (
    InvoiceTypeA InvoiceType = "A"
    InvoiceTypeB InvoiceType = "B"
    InvoiceTypeC InvoiceType = "C"
)

func ParseInvoiceType(s string) (InvoiceType, error)
func (t InvoiceType) String() string
func (t InvoiceType) Valid() bool
```

| Type | ARCA Code | Description |
|------|-----------|-------------|
| A | 1 | Responsable Inscripto → Responsable Inscripto |
| B | 6 | Responsable Inscripto → Consumidor Final/Monotributista |
| C | 11 | Monotributista/Exento → Consumidor Final |

---

### Customer

**Location:** `internal/domain/customer/customer.go`

**Entity:** SaaS customer (invoice issuer).

```go
type Customer struct {
    ID           string
    CUIT         string
    Name         string
    Email        string
    Address      string
    IVACondition IVACondition
    CountryCode  string
    CreatedAt    string
    UpdatedAt    string
}

func NewCustomer(cuit, name, email, address string, ivaCondition IVACondition, countryCode string) (*Customer, error)
```

**Validation:**
- CUIT: 11 digits + checksum (Modulo 11)
- Name: required
- Email: RFC5322 format (if provided)
- Address: required
- IVA Condition: valid enum
- Country Code: ISO 3166-1 alpha-2 (default "AR")

---

### IVACondition

**Location:** `internal/domain/customer/customer.go`

**Enumeration:** AFIP VAT responsibility.

```go
type IVACondition string

const (
    IVAConditionRI IVACondition = "RI" // Responsable Inscripto
    IVAConditionMT IVACondition = "MT" // Monotributista
    IVAConditionEX IVACondition = "EX" // Exento
    IVAConditionCF IVACondition = "CF" // Consumidor Final
    IVAConditionNC IVACondition = "NC" // No Categorizado
)

func ParseIVACondition(s string) (IVACondition, error)
func (c IVACondition) String() string
func (c IVACondition) Valid() bool
```

---

### ARCACredential

**Location:** `internal/domain/credential/credential.go`

**Entity:** Certificate + private key for ARCA.

```go
type ARCACredential struct {
    ID           string
    CustomerID   string
    Environment  Environment       // homologation, production
    CUIT         string
    CertPEM      []byte
    KeyPEM       []byte
    ExpiresAt    time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

func NewARCACredential(customerID string, environment Environment, cuit string, certPEM, keyPEM []byte, expiresAt time.Time) (*ARCACredential, error)

func (c *ARCACredential) IsExpired() bool
func (c *ARCACredential) DaysUntilExpiry() int
```

**Validation:**
- CustomerID required
- Environment valid
- CUIT required
- CertPEM + KeyPEM required (non-empty)
- ExpiresAt in future

---

### Environment

**Location:** `internal/domain/credential/credential.go`

**Enumeration:** ARCA environment.

```go
type Environment string

const (
    EnvironmentHomologation Environment = "homologation"
    EnvironmentProduction   Environment = "production"
)

func ParseEnvironment(s string) (Environment, error)
func (e Environment) String() string
func (e Environment) Valid() bool
```

---

### APIKey

**Location:** `internal/domain/apikey/apikey.go`

**Entity:** Authentication credential.

```go
type APIKey struct {
    ID          string
    CustomerID  string
    Name        string
    Prefix      string       // sk_live_, sk_test_
    Hash        string       // HMAC-SHA256(pepper, raw)
    LastFour    string       // Last 4 chars of raw
    Scopes      []string
    ExpiresAt   *time.Time
    RevokedAt   *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func NewAPIKey(customerID, name, prefix, pepper string, scopes []string, expiresAt *time.Time) (*APIKey, string, error)
func GenerateAPIKey(prefix string) (string, string, error)
func HashAPIKey(key, pepper string) string
func (k *APIKey) Validate(rawKey, pepper string) error
func (k *APIKey) Revoke()
func (k *APIKey) IsRevoked() bool
func (k *APIKey) IsExpired() bool
func (k *APIKey) Masked() string
func ParseAPIKey(rawKey string) (prefix, suffix string, error)
```

**Security:**
- Raw key = prefix + 64 hex chars (32 bytes entropy)
- Hash = HMAC-SHA256(pepper, rawKey)
- Constant-time comparison
- Pepper from env var `API_KEY_PEPPER`

---

### IdempotencyRecord

**Location:** `internal/domain/idempotency/idempotency.go`

**Entity:** Deduplication state.

```go
type IdempotencyRecord struct {
    Key         string
    RequestHash string
    Status      IdempotencyStatus
    Response    []byte
    Error       string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ExpiresAt   time.Time
}

func NewIdempotencyRecord(key string, requestBody []byte, ttl time.Duration) (*IdempotencyRecord, error)
func (r *IdempotencyRecord) CheckConflict(requestBody []byte) error
func (r *IdempotencyRecord) Complete(response []byte)
func (r *IdempotencyRecord) Fail(err error)
func (r *IdempotencyRecord) IsExpired() bool
```

**Status Enum:**
```go
type IdempotencyStatus string

const (
    StatusProcessing IdempotencyStatus = "processing"
    StatusSucceeded  IdempotencyStatus = "succeeded"
    StatusFailed     IdempotencyStatus = "failed"
)
```

---

## HTTP DTO Contracts

### IssueInvoiceRequest/Response

**Location:** `internal/interfaces/http/dto/dto.go`

```go
type IssueInvoiceRequest struct {
    CUIT           string                      `json:"cuit" validate:"required,len=11"`
    InvoiceType    string                      `json:"tipo_factura" validate:"required,oneof=A B C"`
    PointOfSale    int                         `json:"punto_venta" validate:"required,min=1"`
    Items          []IssueInvoiceItemRequest   `json:"items" validate:"required,min=1,dive"`
    IdempotencyKey string                      `json:"-" header:"Idempotency-Key"`
}

type IssueInvoiceItemRequest struct {
    Description    string `json:"descripcion" validate:"required"`
    Quantity       int    `json:"cantidad" validate:"required,min=1"`
    UnitPriceCents int64  `json:"precio_unitario" validate:"required,min=0"`
}

type IssueInvoiceResponse struct {
    InvoiceID      string                  `json:"invoice_id"`
    Status         string                  `json:"status"`
    InvoiceType    string                  `json:"tipo_factura"`
    PointOfSale    int                     `json:"punto_venta"`
    CAE            string                  `json:"cae,omitempty"`
    CAEExpiresAt   *time.Time              `json:"cae_expires_at,omitempty"`
    VoucherNumber  int64                   `json:"numero_comprobante,omitempty"`
    VoucherType    int                     `json:"tipo_comprobante,omitempty"`
    SubtotalCents  int64                   `json:"subtotal_cents"`
    TaxCents       int64                   `json:"tax_cents"`
    TotalCents     int64                   `json:"total_cents"`
    Items          []InvoiceItemResponse   `json:"items"`
    IssuedAt       *time.Time              `json:"issued_at,omitempty"`
    CreatedAt      time.Time               `json:"created_at"`
}
```

---

### APIKey DTOs

```go
type APIKeyRequest struct {
    Name      string   `json:"name" validate:"required"`
    Prefix    string   `json:"prefix" validate:"required,oneof=sk_live_ sk_test_"`
    Scopes    []string `json:"scopes"`
    ExpiresAt *int64   `json:"expires_at,omitempty"`
}

type APIKeyResponse struct {
    ID        string   `json:"id"`
    Name      string   `json:"name"`
    Prefix    string   `json:"prefix"`
    LastFour  string   `json:"last_four"`
    Scopes    []string `json:"scopes"`
    ExpiresAt *int64   `json:"expires_at,omitempty"`
    RevokedAt *int64   `json:"revoked_at,omitempty"`
    CreatedAt int64    `json:"created_at"`
}

type APIKeyCreateResponse struct {
    APIKey string        `json:"api_key"`  // Only returned once
    Details APIKeyResponse `json:"details"`
}
```

---

### ErrorResponse

```go
type ErrorResponse struct {
    Error   string            `json:"error"`
    Code    string            `json:"code"`
    Details map[string]string `json:"details,omitempty"`
}
```

**Standard Error Codes:**
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_request` | 400 | Malformed JSON |
| `invalid_cuit` | 400 | CUIT format/checksum |
| `invalid_invoice_type` | 400 | Not A/B/C |
| `invalid_amount` | 400 | Item validation |
| `invalid_api_key` | 401 | Missing/invalid auth |
| `api_key_revoked` | 403 | Key revoked |
| `api_key_expired` | 403 | Key expired |
| `idempotency_conflict` | 409 | Same key, different payload |
| `idempotency_processing` | 409 | Request in progress |
| `arca_rejected` | 422 | ARCA rejection |
| `arca_service_unavailable` | 503 | ARCA unreachable |
| `database_error` | 500 | DB failure |
| `internal_error` | 500 | Unexpected |

---

## Contract Evolution Guidelines

### Versioning
- API version in URL: `/v1/`
- Breaking changes → `/v2/`
- Additive changes backward compatible

### Adding Fields
- Optional fields with `omitempty`
- New enum values documented
- Default values for new required fields

### Deprecation
- Mark in OpenAPI with `deprecated: true`
- 6-month sunset period
- Communicate via headers + docs

### Testing Contracts
- Consumer-driven contract tests (Pact)
- Schema validation in CI
- Mock server for integration tests