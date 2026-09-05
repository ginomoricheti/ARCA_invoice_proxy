package tests

import (
	"context"
	"testing"
	"time"

	"arca-invoice-proxy/internal/application/authentication"
	"arca-invoice-proxy/internal/domain/apikey"
	"arca-invoice-proxy/internal/domain/errors"
)

// Mock store
type mockAPIKeyStore struct {
	keys map[string]*apikey.APIKey
}

func newMockAPIKeyStore() *mockAPIKeyStore {
	return &mockAPIKeyStore{keys: make(map[string]*apikey.APIKey)}
}

func (m *mockAPIKeyStore) Create(ctx context.Context, key *apikey.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockAPIKeyStore) GetByPrefixAndSuffix(ctx context.Context, prefix, suffix string) (*apikey.APIKey, error) {
	for _, k := range m.keys {
		if k.Prefix == prefix && k.LastFour == suffix {
			return k, nil
		}
	}
	return nil, errors.New(errors.CodeNotFound, "not found")
}

func (m *mockAPIKeyStore) GetByID(ctx context.Context, id string) (*apikey.APIKey, error) {
	if k, ok := m.keys[id]; ok {
		return k, nil
	}
	return nil, errors.New(errors.CodeNotFound, "not found")
}

func (m *mockAPIKeyStore) ListByCustomer(ctx context.Context, customerID string) ([]*apikey.APIKey, error) {
	var result []*apikey.APIKey
	for _, k := range m.keys {
		if k.CustomerID == customerID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockAPIKeyStore) Update(ctx context.Context, key *apikey.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func TestAuthService_ValidateAPIKey(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	// Create a test key
	expiresAt := time.Now().Add(24 * time.Hour)
	key, rawKey, _ := apikey.NewAPIKey("cust_123", "Test Key", apikey.PrefixLive, "test-pepper", []string{"invoices:write"}, &expiresAt)
	store.keys[key.ID] = key

	tests := []struct {
		name         string
		rawKey       string
		shouldPass   bool
		expectedCode errors.ErrorCode
	}{
		{"valid key", rawKey, true, ""},
		{"wrong key", rawKey + "x", false, errors.CodeInvalidAPIKey},
		{"empty key", "", false, errors.CodeInvalidAPIKey},
		{"invalid format", "invalid", false, errors.CodeInvalidAPIKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ValidateAPIKey(context.Background(), tt.rawKey)
			if tt.shouldPass {
				if err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if result == nil {
					t.Error("result should not be nil")
				}
				if result.ID != key.ID {
					t.Error("wrong key returned")
				}
			} else {
				if err == nil {
					t.Error("expected error")
				}
				if !errors.IsCode(err, tt.expectedCode) {
					t.Errorf("expected code %s, got %s", tt.expectedCode, errors.GetCode(err))
				}
			}
		})
	}
}

func TestAuthService_ValidateAPIKey_Revoked(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	key, rawKey, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "test-pepper", []string{}, nil)
	key.Revoke()
	store.keys[key.ID] = key

	_, err := svc.ValidateAPIKey(context.Background(), rawKey)
	if err == nil {
		t.Error("expected error for revoked key")
	}
	if !errors.IsCode(err, errors.CodeAPIKeyRevoked) {
		t.Errorf("expected CodeAPIKeyRevoked, got %s", errors.GetCode(err))
	}
}

func TestAuthService_ValidateAPIKey_Expired(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	past := time.Now().Add(-1 * time.Hour)
	key, rawKey, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "test-pepper", []string{}, &past)
	store.keys[key.ID] = key

	_, err := svc.ValidateAPIKey(context.Background(), rawKey)
	if err == nil {
		t.Error("expected error for expired key")
	}
	if !errors.IsCode(err, errors.CodeAPIKeyExpired) {
		t.Errorf("expected CodeAPIKeyExpired, got %s", errors.GetCode(err))
	}
}

func TestAuthService_CreateAPIKey(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	key, rawKey, err := svc.CreateAPIKey(context.Background(), "cust_123", "New Key", apikey.PrefixTest, []string{"invoices:read"}, &expiresAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key == nil {
		t.Fatal("key should not be nil")
	}
	if key.CustomerID != "cust_123" {
		t.Error("customer ID mismatch")
	}
	if key.Name != "New Key" {
		t.Error("name mismatch")
	}
	if key.Prefix != apikey.PrefixTest {
		t.Error("prefix mismatch")
	}
	if len(key.Scopes) != 1 || key.Scopes[0] != "invoices:read" {
		t.Error("scopes mismatch")
	}
	if key.ExpiresAt == nil {
		t.Error("expires at should be set")
	}
	if rawKey == "" {
		t.Error("raw key should not be empty")
	}
	if len(rawKey) != len(apikey.PrefixTest)+apikey.KeyLength*2 {
		t.Error("raw key length mismatch")
	}
}

func TestAuthService_ListAPIKeys(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	key1, _, _ := apikey.NewAPIKey("cust_123", "Key 1", apikey.PrefixLive, "test-pepper", []string{}, nil)
	key2, _, _ := apikey.NewAPIKey("cust_123", "Key 2", apikey.PrefixTest, "test-pepper", []string{}, nil)
	key3, _, _ := apikey.NewAPIKey("other_cust", "Key 3", apikey.PrefixLive, "test-pepper", []string{}, nil)

	store.keys[key1.ID] = key1
	store.keys[key2.ID] = key2
	store.keys[key3.ID] = key3

	keys, err := svc.ListAPIKeys(context.Background(), "cust_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestAuthService_RevokeAPIKey(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	key, _, _ := apikey.NewAPIKey("cust_123", "Test", apikey.PrefixLive, "test-pepper", []string{}, nil)
	store.keys[key.ID] = key

	err := svc.RevokeAPIKey(context.Background(), key.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	revokedKey := store.keys[key.ID]
	if !revokedKey.IsRevoked() {
		t.Error("key should be revoked")
	}
	if revokedKey.RevokedAt == nil {
		t.Error("revoked at should be set")
	}
}

func TestAuthService_RevokeAPIKey_NotFound(t *testing.T) {
	store := newMockAPIKeyStore()
	svc := authentication.NewService(store, "test-pepper")

	err := svc.RevokeAPIKey(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
	if !errors.IsCode(err, errors.CodeNotFound) {
		t.Errorf("expected CodeNotFound, got %s", errors.GetCode(err))
	}
}
