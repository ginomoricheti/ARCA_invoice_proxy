package authentication

import (
	"context"
	"errors"

	"arca-invoice-proxy/internal/domain/apikey"
	"arca-invoice-proxy/internal/domain/errors"
)

type APIKeyStore interface {
	GetByPrefixAndSuffix(ctx context.Context, prefix, suffix string) (*apikey.APIKey, error)
	GetByID(ctx context.Context, id string) (*apikey.APIKey, error)
	ListByCustomer(ctx context.Context, customerID string) ([]*apikey.APIKey, error)
	Create(ctx context.Context, key *apikey.APIKey) error
	Update(ctx context.Context, key *apikey.APIKey) error
}

type Service struct {
	store  APIKeyStore
	pepper string
}

func NewService(store APIKeyStore, pepper string) *Service {
	return &Service{
		store:  store,
		pepper: pepper,
	}
}

func (s *Service) ValidateAPIKey(ctx context.Context, rawKey string) (*apikey.APIKey, error) {
	prefix, suffix, err := apikey.ParseAPIKey(rawKey)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidAPIKey, "invalid api key format")
	}

	key, err := s.store.GetByPrefixAndSuffix(ctx, prefix, suffix)
	if err != nil {
		if errors.IsCode(err, errors.CodeNotFound) {
			return nil, errors.New(errors.CodeInvalidAPIKey, "invalid api key")
		}
		return nil, err
	}

	if err := key.Validate(rawKey, s.pepper); err != nil {
		switch {
		case errors.Is(err, apikey.ErrKeyRevoked):
			return nil, errors.New(errors.CodeAPIKeyRevoked, "api key has been revoked")
		case errors.Is(err, apikey.ErrKeyExpired):
			return nil, errors.New(errors.CodeAPIKeyExpired, "api key has expired")
		default:
			return nil, errors.New(errors.CodeInvalidAPIKey, "invalid api key")
		}
	}

	return key, nil
}

func (s *Service) CreateAPIKey(ctx context.Context, customerID, name, prefix string, scopes []string, expiresAt *int64) (*apikey.APIKey, string, error) {
	var exp *time.Time
	if expiresAt != nil {
		t := time.Unix(*expiresAt, 0).UTC()
		exp = &t
	}

	key, rawKey, err := apikey.NewAPIKey(customerID, name, prefix, s.pepper, scopes, exp)
	if err != nil {
		return nil, "", errors.Wrap(err, errors.CodeValidationFailed, "create api key")
	}

	if err := s.store.Create(ctx, key); err != nil {
		return nil, "", errors.Wrap(err, errors.CodeDatabaseError, "store api key")
	}

	return key, rawKey, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, customerID string) ([]*apikey.APIKey, error) {
	keys, err := s.store.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "list api keys")
	}
	return keys, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, keyID string) error {
	key, err := s.store.GetByID(ctx, keyID)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "get api key")
	}
	key.Revoke()
	if err := s.store.Update(ctx, key); err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "revoke api key")
	}
	return nil
}