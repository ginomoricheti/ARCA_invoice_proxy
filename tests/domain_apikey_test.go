package tests

import (
	"testing"
	"time"

	"arca-invoice-proxy/internal/domain/apikey"
)

func TestAPIKey_Generate(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		hasError bool
	}{
		{"live prefix", apikey.PrefixLive, false},
		{"test prefix", apikey.PrefixTest, false},
		{"invalid prefix", "invalid_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, suffix, err := apikey.GenerateAPIKey(tt.prefix)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(key) != len(tt.prefix)+apikey.KeyLength*2 {
					t.Errorf("key length mismatch")
				}
				if len(suffix) != apikey.KeyLength*2 {
					t.Errorf("suffix length mismatch")
				}
			}
		})
	}
}

func TestAPIKey_Hash(t *testing.T) {
	key := "sk_live_REDACTED_KEY_FOR_TESTING_abcdef1234567890"
	pepper := "test-pepper"

	hash1 := apikey.HashAPIKey(key, pepper)
	hash2 := apikey.HashAPIKey(key, pepper)

	if hash1 != hash2 {
		t.Error("hash should be deterministic")
	}
	if len(hash1) != 64 {
		t.Errorf("hash length should be 64, got %d", len(hash1))
	}

	// Different key should produce different hash
	hash3 := apikey.HashAPIKey(key+"x", pepper)
	if hash1 == hash3 {
		t.Error("different keys should produce different hashes")
	}
}

func TestAPIKey_Creation(t *testing.T) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	key, rawKey, err := apikey.NewAPIKey("cust_123", "Test Key", apikey.PrefixLive, "pepper", []string{"invoices:write"}, &expiresAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key.CustomerID != "cust_123" {
		t.Error("customer ID mismatch")
	}
	if key.Name != "Test Key" {
		t.Error("name mismatch")
	}
	if key.Prefix != apikey.PrefixLive {
		t.Error("prefix mismatch")
	}
	if len(key.Hash) != 64 {
		t.Error("hash length mismatch")
	}
	if len(key.LastFour) != 4 {
		t.Error("last four length mismatch")
	}
	if key.Scopes == nil {
		t.Error("scopes should not be nil")
	}
	if key.ExpiresAt == nil {
		t.Error("expires at should be set")
	}
	if key.RevokedAt != nil {
		t.Error("revoked at should be nil")
	}
	if rawKey == "" {
		t.Error("raw key should not be empty")
	}
	if !key.Validate(rawKey, "pepper") {
		t.Error("valid key should pass validation")
	}
}

func TestAPIKey_Validation(t *testing.T) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	key, rawKey, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "pepper", []string{}, &expiresAt)

	tests := []struct {
		name        string
		inputKey    string
		inputPepper string
		shouldPass  bool
	}{
		{"correct key and pepper", rawKey, "pepper", true},
		{"wrong key", rawKey + "x", "pepper", false},
		{"wrong pepper", rawKey, "wrong-pepper", false},
		{"empty key", "", "pepper", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := key.Validate(tt.inputKey, tt.inputPepper)
			if tt.shouldPass {
				if err != nil {
					t.Errorf("expected validation to pass, got: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected validation to fail")
				}
			}
		})
	}
}

func TestAPIKey_Revoked(t *testing.T) {
	key, rawKey, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "pepper", []string{}, nil)

	if key.IsRevoked() {
		t.Error("new key should not be revoked")
	}

	key.Revoke()

	if !key.IsRevoked() {
		t.Error("revoked key should be revoked")
	}

	err := key.Validate(rawKey, "pepper")
	if err == nil {
		t.Error("revoked key should fail validation")
	}
}

func TestAPIKey_Expired(t *testing.T) {
	past := time.Now().UTC().Add(-1 * time.Hour)
	key, rawKey, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "pepper", []string{}, &past)

	if !key.IsExpired() {
		t.Error("past expiration should be expired")
	}

	err := key.Validate(rawKey, "pepper")
	if err == nil {
		t.Error("expired key should fail validation")
	}
}

func TestAPIKey_Parse(t *testing.T) {
	tests := []struct {
		input     string
		prefix    string
		suffix    string
		hasError  bool
	}{
		{"sk_live_REDACTED_KEY_FOR_TESTING_abcdef1234567890", apikey.PrefixLive, "REDACTED_KEY_FOR_TESTING_abcdef1234567890", false},
		{"sk_test_REDACTED_KEY_FOR_TESTING_abcdef1234567890", apikey.PrefixTest, "REDACTED_KEY_FOR_TESTING_abcdef1234567890", false},
		{"invalid_key", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			prefix, suffix, err := apikey.ParseAPIKey(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if prefix != tt.prefix {
					t.Errorf("prefix mismatch")
				}
				if suffix != tt.suffix {
					t.Errorf("suffix mismatch")
				}
			}
		})
	}
}

func TestAPIKey_Masked(t *testing.T) {
	key, _, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "pepper", []string{}, nil)
	masked := key.Masked()

	if len(masked) != len(key.Prefix)+len(key.Hash)-8+4 {
		t.Error("masked length mismatch")
	}
	if masked[:len(key.Prefix)] != key.Prefix {
		t.Error("masked should start with prefix")
	}
	if masked[len(masked)-4:] != key.LastFour {
		t.Error("masked should end with last four")
	}
}
