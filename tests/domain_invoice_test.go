package tests

import (
	"testing"
	"time"

	"arca-invoice-proxy/internal/domain/invoice"
)

func TestInvoiceType_Parse(t *testing.T) {
	tests := []struct {
		input    string
		expected invoice.InvoiceType
		hasError bool
	}{
		{"A", invoice.InvoiceTypeA, false},
		{"B", invoice.InvoiceTypeB, false},
		{"C", invoice.InvoiceTypeC, false},
		{"D", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := invoice.ParseInvoiceType(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestInvoiceType_Valid(t *testing.T) {
	tests := []struct {
		input    invoice.InvoiceType
		expected bool
	}{
		{invoice.InvoiceTypeA, true},
		{invoice.InvoiceTypeB, true},
		{invoice.InvoiceTypeC, true},
		{invoice.InvoiceType("D"), false},
		{invoice.InvoiceType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.Valid() != tt.expected {
				t.Errorf("expected Valid()=%v for %q", tt.expected, tt.input)
			}
		})
	}
}

func TestInvoiceItem_Creation(t *testing.T) {
	tests := []struct {
		name           string
		description    string
		quantity       int
		unitPriceCents int64
		hasError       bool
	}{
		{"valid", "Servicio", 1, 100000, false},
		{"empty description", "", 1, 100000, true},
		{"zero quantity", "Servicio", 0, 100000, true},
		{"negative quantity", "Servicio", -1, 100000, true},
		{"negative price", "Servicio", 1, -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := invoice.NewInvoiceItem(tt.description, tt.quantity, tt.unitPriceCents)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if item.Description != tt.description {
					t.Errorf("description mismatch")
				}
				if item.Quantity != tt.quantity {
					t.Errorf("quantity mismatch")
				}
				if item.UnitPriceCents != tt.unitPriceCents {
					t.Errorf("unit price mismatch")
				}
				expectedTotal := int64(tt.quantity) * tt.unitPriceCents
				if item.TotalCents() != expectedTotal {
					t.Errorf("total cents: expected %d, got %d", expectedTotal, item.TotalCents())
				}
			}
		})
	}
}

func TestInvoice_Creation(t *testing.T) {
	items := []invoice.InvoiceItem{
		{Description: "Item 1", Quantity: 2, UnitPriceCents: 50000},
		{Description: "Item 2", Quantity: 1, UnitPriceCents: 100000},
	}

	inv, err := invoice.NewInvoice("cust_123", invoice.InvoiceTypeC, 1, items, "idem_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.CustomerID != "cust_123" {
		t.Errorf("customer ID mismatch")
	}
	if inv.Type != invoice.InvoiceTypeC {
		t.Errorf("invoice type mismatch")
	}
	if inv.PointOfSale != 1 {
		t.Errorf("point of sale mismatch")
	}
	if len(inv.Items) != 2 {
		t.Errorf("items count mismatch")
	}
	if inv.IdempotencyKey != "idem_123" {
		t.Errorf("idempotency key mismatch")
	}
	if inv.Status != invoice.InvoiceStatusPending {
		t.Errorf("status should be pending")
	}
	if inv.SubtotalCents != 200000 {
		t.Errorf("subtotal: expected 200000, got %d", inv.SubtotalCents)
	}
	if inv.TotalCents != 200000 {
		t.Errorf("total: expected 200000, got %d", inv.TotalCents)
	}
	if inv.ID == "" {
		t.Error("ID should be generated")
	}
}

func TestInvoice_Creation_Errors(t *testing.T) {
	tests := []struct {
		name        string
		items       []invoice.InvoiceItem
		pointOfSale int
		invType     invoice.InvoiceType
		hasError    bool
	}{
		{"empty items", []invoice.InvoiceItem{}, 1, invoice.InvoiceTypeC, true},
		{"zero point of sale", []invoice.InvoiceItem{{Description: "A", Quantity: 1, UnitPriceCents: 100}}, 0, invoice.InvoiceTypeC, true},
		{"invalid type", []invoice.InvoiceItem{{Description: "A", Quantity: 1, UnitPriceCents: 100}}, 1, invoice.InvoiceType("X"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invoice.NewInvoice("cust_123", tt.invType, tt.pointOfSale, tt.items, "idem_123")
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInvoice_MarkIssued(t *testing.T) {
	items := []invoice.InvoiceItem{{Description: "A", Quantity: 1, UnitPriceCents: 100}}
	inv, _ := invoice.NewInvoice("cust_123", invoice.InvoiceTypeC, 1, items, "idem_123")

	response := invoice.ARCAResponse{
		CAE:               "12345678901234",
		CAEExpirationDate: time.Now().UTC().AddDate(0, 0, 10),
		VoucherNumber:     1,
		VoucherType:       11,
		Result:            "A",
		Observations:      []string{},
	}

	inv.MarkIssued(response)

	if inv.Status != invoice.InvoiceStatusIssued {
		t.Errorf("status should be issued")
	}
	if inv.ARCAResponse == nil {
		t.Error("ARCA response should be set")
	}
	if inv.ARCAResponse.CAE != "12345678901234" {
		t.Error("CAE mismatch")
	}
	if inv.IssuedAt == nil {
		t.Error("IssuedAt should be set")
	}
}

func TestInvoice_MarkRejected(t *testing.T) {
	items := []invoice.InvoiceItem{{Description: "A", Quantity: 1, UnitPriceCents: 100}}
	inv, _ := invoice.NewInvoice("cust_123", invoice.InvoiceTypeC, 1, items, "idem_123")

	inv.MarkRejected([]string{"Error 1", "Error 2"})

	if inv.Status != invoice.InvoiceStatusRejected {
		t.Errorf("status should be rejected")
	}
	if inv.ARCAResponse == nil {
		t.Error("ARCA response should be set")
	}
	if inv.ARCAResponse.Result != "rejected" {
		t.Error("result should be rejected")
	}
	if len(inv.ARCAResponse.Observations) != 2 {
		t.Error("observations should have 2 items")
	}
}

func TestInvoice_MarkCancelled(t *testing.T) {
	items := []invoice.InvoiceItem{{Description: "A", Quantity: 1, UnitPriceCents: 100}}
	inv, _ := invoice.NewInvoice("cust_123", invoice.InvoiceTypeC, 1, items, "idem_123")

	inv.MarkCancelled()

	if inv.Status != invoice.InvoiceStatusCancelled {
		t.Errorf("status should be cancelled")
	}
}
