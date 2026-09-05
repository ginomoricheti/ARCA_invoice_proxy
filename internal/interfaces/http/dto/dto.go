package dto

import (
	"time"

	"arca-invoice-proxy/internal/domain/invoice"
)

type IssueInvoiceRequest struct {
	CUIT           string                    `json:"cuit" validate:"required,len=11"`
	InvoiceType    string                    `json:"tipo_factura" validate:"required,oneof=A B C"`
	PointOfSale    int                       `json:"punto_venta" validate:"required,min=1"`
	Items          []IssueInvoiceItemRequest `json:"items" validate:"required,min=1,dive"`
	IdempotencyKey string                    `json:"-" header:"Idempotency-Key"`
}

type IssueInvoiceItemRequest struct {
	Description    string `json:"descripcion" validate:"required"`
	Quantity       int    `json:"cantidad" validate:"required,min=1"`
	UnitPriceCents int64  `json:"precio_unitario" validate:"required,min=0"`
}

type IssueInvoiceResponse struct {
	InvoiceID     string                `json:"invoice_id"`
	Status        string                `json:"status"`
	InvoiceType   string                `json:"tipo_factura"`
	PointOfSale   int                   `json:"punto_venta"`
	CAE           string                `json:"cae,omitempty"`
	CAEExpiresAt  *time.Time            `json:"cae_expires_at,omitempty"`
	VoucherNumber int64                 `json:"numero_comprobante,omitempty"`
	VoucherType   int                   `json:"tipo_comprobante,omitempty"`
	SubtotalCents int64                 `json:"subtotal_cents"`
	TaxCents      int64                 `json:"tax_cents"`
	TotalCents    int64                 `json:"total_cents"`
	Items         []InvoiceItemResponse `json:"items"`
	IssuedAt      *time.Time            `json:"issued_at,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
}

type InvoiceItemResponse struct {
	Description    string `json:"descripcion"`
	Quantity       int    `json:"cantidad"`
	UnitPriceCents int64  `json:"precio_unitario"`
	TotalCents     int64  `json:"total_cents"`
}

type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code"`
	Details map[string]string `json:"details,omitempty"`
}

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

func ToInvoiceItemResponse(item invoice.InvoiceItem) InvoiceItemResponse {
	return InvoiceItemResponse{
		Description:    item.Description,
		Quantity:       item.Quantity,
		UnitPriceCents: item.UnitPriceCents,
		TotalCents:     item.TotalCents(),
	}
}
