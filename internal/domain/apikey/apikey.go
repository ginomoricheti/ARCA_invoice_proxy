package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidAPIKey     = errors.New("invalid API key")
	ErrKeyNotFound       = errors.New("API key not found")
	ErrKeyRevoked        = errors.New("API key has been revoked")
	ErrKeyExpired        = errors.New("API key has expired")
	ErrInvalidPrefix     = errors.New("invalid API key prefix")
	ErrInvalidFormat     = errors.New("invalid API key format")
)

const (
	PrefixLive    = "sk_live_"
	PrefixTest    = "sk_test_"
	KeyLength     = 32
	HashLength    = 32
)

type APIKey struct {
	ID          string
	CustomerID  string
	Name        string
	Prefix      string
	Hash        string
	LastFour    string
	Scopes      []string
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func GenerateAPIKey(prefix string) (string, string, error) {
	if prefix != PrefixLive && prefix != PrefixTest {
		return "", "", ErrInvalidPrefix
	}

	randomBytes := make([]byte, KeyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}

	key := prefix + hex.EncodeToString(randomBytes)
	return key, key[len(prefix):], nil
}

func HashAPIKey(key, pepper string) string {
	mac := hmac.New(sha256.New, []byte(pepper))
	mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}

func NewAPIKey(customerID, name, prefix, pepper string, scopes []string, expiresAt *time.Time) (*APIKey, string, error) {
	if customerID == "" {
		return nil, "", errors.Join(ErrInvalidAPIKey, errors.New("customer ID is required"))
	}
	if name == "" {
		return nil, "", errors.Join(ErrInvalidAPIKey, errors.New("name is required"))
	}

	rawKey, suffix, err := GenerateAPIKey(prefix)
	if err != nil {
		return nil, "", err
	}

	hash := HashAPIKey(rawKey, pepper)
	lastFour := suffix[len(suffix)-4:]

	now := time.Now().UTC()
	apiKey := &APIKey{
		ID:         generateID(),
		CustomerID: customerID,
		Name:       name,
		Prefix:     prefix,
		Hash:       hash,
		LastFour:   lastFour,
		Scopes:     scopes,
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return apiKey, rawKey, nil
}

func (k *APIKey) Validate(rawKey, pepper string) error {
	if k.RevokedAt != nil {
		return ErrKeyRevoked
	}
	if k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt) {
		return ErrKeyExpired
	}

	expectedHash := HashAPIKey(rawKey, pepper)
	if !hmac.Equal([]byte(k.Hash), []byte(expectedHash)) {
		return ErrInvalidAPIKey
	}
	return nil
}

func (k *APIKey) Revoke() {
	now := time.Now().UTC()
	k.RevokedAt = &now
	k.UpdatedAt = now
}

func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

func (k *APIKey) IsExpired() bool {
	return k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt)
}

func (k *APIKey) Masked() string {
	return k.Prefix + strings.Repeat("*", len(k.Hash)-8) + k.LastFour
}

func ParseAPIKey(rawKey string) (prefix, suffix string, err error) {
	if strings.HasPrefix(rawKey, PrefixLive) {
		return PrefixLive, rawKey[len(PrefixLive):], nil
	}
	if strings.HasPrefix(rawKey, PrefixTest) {
		return PrefixTest, rawKey[len(PrefixTest):], nil
	}
	return "", "", ErrInvalidFormat
}

func generateID() string {
	return "key_" + time.Now().UTC().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

type APIKeyStore interface {
	Create(ctx interface{}, key *APIKey) error
	GetByPrefixAndSuffix(ctx interface{}, prefix, suffix string) (*APIKey, error)
	GetByID(ctx interface{}, id string) (*APIKey, error)
	ListByCustomer(ctx interface{}, customerID string) ([]*APIKey, error)
	Update(ctx interface{}, key *APIKey) error
}