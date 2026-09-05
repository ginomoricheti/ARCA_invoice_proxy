# API Documentation

## Base URL

- Development: `http://localhost:8080`
- Production: `https://api.arca-invoice-proxy.example.com`

## Authentication

All API endpoints (except `/health` and `/ready`) require authentication via **API Key** in the Authorization header:

```
Authorization: Bearer <YOUR_API_KEY_LIVE>
Authorization: Bearer <YOUR_API_KEY_TEST>
```

- `sk_live_` prefix: Production keys
- `sk_test_` prefix: Test/homologation keys

API Keys are managed via the `/v1/api-keys` endpoints.

## Idempotency

The `POST /v1/facturar` endpoint supports idempotency via the `Idempotency-Key` header:

```
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

- Same key + same payload = returns cached response (201)
- Same key + different payload = 409 Conflict
- Key expires after 24 hours (configurable)
- Keys are scoped per user (API Key)

## Endpoints

### Health Check

#### GET /health

Returns service health status.

**Response (200):**
```json
{
  "status": "ok",
  "timestamp": "2026-01-04T12:00:00Z",
  "version": "1.0.0"
}
```

#### GET /ready

Returns service readiness status.

**Response (200):**
```json
{
  "status": "ready"
}
```

---

### Issue Invoice

#### POST /v1/facturar

Creates and issues an electronic invoice through ARCA.

**Headers:**
```
Authorization: Bearer sk_live_xxxxx
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000  (optional)
Content-Type: application/json
```

**Request Body:**
```json
{
  "cuit": "20123456789",
  "tipo_factura": "C",
  "punto_venta": 1,
  "items": [
    {
      "descripcion": "Servicio de desarrollo",
      "cantidad": 1,
      "precio_unitario": 1000000
    }
  ]
}
```

**Field Details:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| cuit | string | Yes | 11-digit CUIT of the issuer |
| tipo_factura | string | Yes | Invoice type: "A", "B", or "C" |
| punto_venta | integer | Yes | Point of sale number (GÎ—1) |
| items | array | Yes | At least one item |
| items[].descripcion | string | Yes | Item description |
| items[].cantidad | integer | Yes | Quantity (GÎ—1) |
| items[].precio_unitario | integer | Yes | Unit price in cents (GÎ—0) |

**Success Response (201):**
```json
{
  "invoice_id": "inv_20260104120000_abc12345",
  "status": "issued",
  "tipo_factura": "C",
  "punto_venta": 1,
  "cae": "12345678901234",
  "cae_expires_at": "2026-01-14T00:00:00Z",
  "numero_comprobante": 1,
  "tipo_comprobante": 11,
  "subtotal_cents": 1000000,
  "tax_cents": 0,
  "total_cents": 1000000,
  "items": [
    {
      "descripcion": "Servicio de desarrollo",
      "cantidad": 1,
      "precio_unitario": 1000000,
      "total_cents": 1000000
    }
  ],
  "issued_at": "2026-01-04T12:00:00Z",
  "created_at": "2026-01-04T12:00:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_request` | Invalid JSON or missing fields |
| 400 | `invalid_cuit` | CUIT format or checksum invalid |
| 400 | `invalid_invoice_type` | Must be A, B, or C |
| 400 | `invalid_amount` | Invalid item quantity or price |
| 401 | `invalid_api_key` | Missing or invalid Authorization header |
| 401 | `api_key_revoked` | API key has been revoked |
| 401 | `api_key_expired` | API key has expired |
| 409 | `idempotency_conflict` | Same key, different payload |
| 409 | `idempotency_processing` | Request still processing |
| 422 | `arca_rejected` | ARCA rejected the invoice |
| 500 | `internal_error` | Unexpected server error |
| 503 | `arca_service_unavailable` | ARCA service unreachable |

---

### API Key Management

#### POST /v1/api-keys

Creates a new API key.

**Headers:**
```
Authorization: Bearer sk_live_xxxxx
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Production Integration",
  "prefix": "sk_live_",
  "scopes": ["invoices:write", "invoices:read"],
  "expires_at": 1893456000
}
```

**Field Details:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Descriptive name (1-100 chars) |
| prefix | string | Yes | "sk_live_" or "sk_test_" |
| scopes | array | No | Permission scopes |
| expires_at | integer | No | Unix timestamp for expiration |

**Response (201):**
```json
{
  "api_key": "<YOUR_API_KEY>",
  "details": {
    "id": "key_20260104120000_abc12345",
    "name": "Production Integration",
    "prefix": "sk_live_",
    "last_four": "7890",
    "scopes": ["invoices:write", "invoices:read"],
    "expires_at": 1893456000,
    "revoked_at": null,
    "created_at": 1704374400
  }
}
```

> **Important**: The full `api_key` is only returned once at creation. Store it securely.

#### GET /v1/api-keys

Lists all API keys for the authenticated user.

**Headers:**
```
Authorization: Bearer sk_live_xxxxx
```

**Response (200):**
```json
[
  {
    "id": "key_20260104120000_abc12345",
    "name": "Production Integration",
    "prefix": "sk_live_",
    "last_four": "7890",
    "scopes": ["invoices:write", "invoices:read"],
    "expires_at": 1893456000,
    "revoked_at": null,
    "created_at": 1704374400
  },
  {
    "id": "key_20260104120000_def67890",
    "name": "Development Integration",
    "prefix": "sk_test_",
    "last_four": "8901",
    "scopes": ["invoices:write"],
    "expires_at": null,
    "revoked_at": null,
    "created_at": 1704374500
  }
]
```

#### DELETE /v1/api-keys/{id}

Revokes an API key.

**Headers:**
```
Authorization: Bearer sk_live_xxxxx
```

**Path Parameters:**
- `id`: API Key ID

**Response (204):** No content

---

## Error Format

All error responses follow this format:

```json
{
  "error": "invalid_cuit: invalid CUIT",
  "code": "invalid_cuit",
  "details": {
    "field": "cuit",
    "reason": "checksum mismatch"
  }
}
```

## Invoice Types

| Type | Description | ARCA Code | Use Case |
|------|-------------|-----------|----------|
| A | Factura A | 1 | Responsable Inscripto GÂ∆ Responsable Inscripto |
| B | Factura B | 6 | Responsable Inscripto GÂ∆ Consumidor Final / Monotributista |
| C | Factura C | 11 | Monotributista / Exento GÂ∆ Consumidor Final |

## Amounts

All monetary amounts are in **cents** (integer) to avoid floating-point precision issues.

- $1,000.00 GÂ∆ `100000`
- $100.50 GÂ∆ `10050`
- $0.01 GÂ∆ `1`

## Rate Limiting

Rate limiting is not implemented in the MVP but should be configured at the reverse proxy level (nginx, Traefik, etc.).

Recommended limits:
- 100 requests/minute per API key for `/v1/facturar`
- 1000 requests/minute per API key for other endpoints

## Versioning

API version is in the URL path: `/v1/`

Breaking changes will result in a new version (`/v2/`).

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at `docs/openapi.yaml` and can be viewed with tools like Swagger UI or Redoc.
