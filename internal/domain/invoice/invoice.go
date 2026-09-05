package invoice

import (
	"errors"
	"time"
)

type InvoiceType string

const (
	InvoiceTypeA InvoiceType = "A"
	InvoiceTypeB InvoiceType = "B"
	InvoiceTypeC InvoiceType = "C"
)

var (
	ErrInvalidInvoiceType = errors.New("invalid invoice type")
	ErrEmptyItems         = errors.New("invoice must have at least one item")
	ErrInvalidItem        = errors.New("invalid invoice item")
)

func ParseInvoiceType(s string) (InvoiceType, error) {
	switch s {
	case "A":
		return InvoiceTypeA, nil
	case "B":
		return InvoiceTypeB, nil
	case "C":
		return InvoiceTypeC, nil
	default:
		return "", ErrInvalidInvoiceType
	}
}

func (t InvoiceType) String() string {
	return string(t)
}

func (t InvoiceType) Valid() bool {
	return t == InvoiceTypeA || t == InvoiceTypeB || t == InvoiceTypeC
}

type InvoiceItem struct {
	Description    string
	Quantity       int
	UnitPriceCents int64
}

func NewInvoiceItem(description string, quantity int, unitPriceCents int64) (InvoiceItem, error) {
	if description == "" {
		return InvoiceItem{}, errors.Join(ErrInvalidItem, errors.New("description is required"))
	}
	if quantity <= 0 {
		return InvoiceItem{}, errors.Join(ErrInvalidItem, errors.New("quantity must be positive"))
	}
	if unitPriceCents < 0 {
		return InvoiceItem{}, errors.Join(ErrInvalidItem, errors.New("unit price cannot be negative"))
	}
	return InvoiceItem{
		Description:    description,
		Quantity:       quantity,
		UnitPriceCents: unitPriceCents,
	}, nil
}

func (i InvoiceItem) TotalCents() int64 {
	return int64(i.Quantity) * i.UnitPriceCents
}

type InvoiceStatus string

const (
	InvoiceStatusPending   InvoiceStatus = "pending"
	InvoiceStatusIssued    InvoiceStatus = "issued"
	InvoiceStatusRejected  InvoiceStatus = "rejected"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

func (s InvoiceStatus) String() string {
	return string(s)
}

func (s InvoiceStatus) Valid() bool {
	return s == InvoiceStatusPending || s == InvoiceStatusIssued || s == InvoiceStatusRejected || s == InvoiceStatusCancelled
}

type Invoice struct {
	ID             string
	CustomerID     string
	Type           InvoiceType
	PointOfSale    int
	Items          []InvoiceItem
	Status         InvoiceStatus
	SubtotalCents  int64
	TaxCents       int64
	TotalCents     int64
	ARCAResponse   *ARCAResponse
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	IssuedAt       *time.Time
}

type ARCAResponse struct {
	CAE               string
	CAEExpirationDate time.Time
	VoucherNumber     int64
	VoucherType       int
	Result            string
	Observations      []string
}

func NewInvoice(
	customerID string,
	invoiceType InvoiceType,
	pointOfSale int,
	items []InvoiceItem,
	idempotencyKey string,
) (*Invoice, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	if pointOfSale <= 0 {
		return nil, errors.New("point of sale must be positive")
	}
	if !invoiceType.Valid() {
		return nil, ErrInvalidInvoiceType
	}

	var subtotal int64
	for _, item := range items {
		subtotal += item.TotalCents()
	}

	now := time.Now().UTC()
	return &Invoice{
		ID:             generateID(),
		CustomerID:     customerID,
		Type:           invoiceType,
		PointOfSale:    pointOfSale,
		Items:          items,
		Status:         InvoiceStatusPending,
		SubtotalCents:  subtotal,
		TaxCents:       0,
		TotalCents:     subtotal,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (i *Invoice) MarkIssued(response ARCAResponse) {
	i.Status = InvoiceStatusIssued
	i.ARCAResponse = &response
	now := time.Now().UTC()
	i.IssuedAt = &now
	i.UpdatedAt = now
}

func (i *Invoice) MarkRejected(observations []string) {
	i.Status = InvoiceStatusRejected
	i.ARCAResponse = &ARCAResponse{
		Result:       "rejected",
		Observations: observations,
	}
	i.UpdatedAt = time.Now().UTC()
}

func (i *Invoice) MarkCancelled() {
	i.Status = InvoiceStatusCancelled
	i.UpdatedAt = time.Now().UTC()
}

func generateID() string {
	return "inv_" + time.Now().UTC().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
