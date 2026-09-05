package tests

import (
	"testing"

	"arca-invoice-proxy/internal/domain/customer"
)

func TestCustomer_New(t *testing.T) {
	tests := []struct {
		name          string
		cuit          string
		name          string
		email         string
		address       string
		ivaCondition  customer.IVACondition
		countryCode   string
		hasError      bool
	}{
		{
			name:         "valid RI",
			cuit:         "20123456789",
			name:         "Test Company",
			email:        "test@example.com",
			address:      "Av. Corrientes 1234",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "AR",
			hasError:     false,
		},
		{
			name:         "valid MT",
			cuit:         "27123456785",
			name:         "Monotributista",
			email:        "mono@example.com",
			address:      "Calle Falsa 123",
			ivaCondition: customer.IVAConditionMT,
			countryCode:  "AR",
			hasError:     false,
		},
		{
			name:         "invalid CUIT format",
			cuit:         "123456789",
			name:         "Test",
			email:        "test@example.com",
			address:      "Address",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "AR",
			hasError:     true,
		},
		{
			name:         "invalid CUIT checksum",
			cuit:         "20123456780",
			name:         "Test",
			email:        "test@example.com",
			address:      "Address",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "AR",
			hasError:     true,
		},
		{
			name:         "empty name",
			cuit:         "20123456789",
			name:         "",
			email:        "test@example.com",
			address:      "Address",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "AR",
			hasError:     true,
		},
		{
			name:         "invalid email",
			cuit:         "20123456789",
			name:         "Test",
			email:        "invalid-email",
			address:      "Address",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "AR",
			hasError:     true,
		},
		{
			name:         "empty address",
			cuit:         "20123456789",
			name:         "Test",
			email:        "test@example.com",
			address:      "",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "AR",
			hasError:     true,
		},
		{
			name:         "invalid IVA",
			cuit:         "20123456789",
			name:         "Test",
			email:        "test@example.com",
			address:      "Address",
			ivaCondition: customer.IVACondition("XX"),
			countryCode:  "AR",
			hasError:     true,
		},
		{
			name:         "invalid country code",
			cuit:         "20123456789",
			name:         "Test",
			email:        "test@example.com",
			address:      "Address",
			ivaCondition: customer.IVAConditionRI,
			countryCode:  "USA",
			hasError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cust, err := customer.NewCustomer(tt.cuit, tt.name, tt.email, tt.address, tt.ivaCondition, tt.countryCode)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cust.CUIT != tt.cuit {
					t.Errorf("CUIT mismatch")
				}
				if cust.Name != tt.name {
					t.Errorf("name mismatch")
				}
				if cust.Email != tt.email {
					t.Errorf("email mismatch")
				}
				if cust.Address != tt.address {
					t.Errorf("address mismatch")
				}
				if cust.IVACondition != tt.ivaCondition {
					t.Errorf("IVA condition mismatch")
				}
				if cust.CountryCode != tt.countryCode {
					t.Errorf("country code mismatch")
				}
			}
		})
	}
}

func TestIVACondition_Parse(t *testing.T) {
	tests := []struct {
		input    string
		expected customer.IVACondition
		hasError bool
	}{
		{"RI", customer.IVAConditionRI, false},
		{"MT", customer.IVAConditionMT, false},
		{"EX", customer.IVAConditionEX, false},
		{"CF", customer.IVAConditionCF, false},
		{"NC", customer.IVAConditionNC, false},
		{"XX", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := customer.ParseIVACondition(tt.input)
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

func TestIVACondition_Valid(t *testing.T) {
	tests := []struct {
		input    customer.IVACondition
		expected bool
	}{
		{customer.IVAConditionRI, true},
		{customer.IVAConditionMT, true},
		{customer.IVAConditionEX, true},
		{customer.IVAConditionCF, true},
		{customer.IVAConditionNC, true},
		{customer.IVACondition("XX"), false},
		{customer.IVACondition(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.Valid() != tt.expected {
				t.Errorf("expected Valid()=%v for %q", tt.expected, tt.input)
			}
		})
	}
}