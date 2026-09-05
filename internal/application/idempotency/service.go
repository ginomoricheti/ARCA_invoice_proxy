package idempotency

import (
	"context"
	"encoding/json"
	"errors"

	"arca-invoice-proxy/internal/domain/errors"
	"arca-invoice-proxy/internal/domain/idempotency"
)

type IdempotencyStore interface {
	Create(ctx context.Context, record *idempotency.IdempotencyRecord) error
	Get(ctx context.Context, key string) (*idempotency.IdempotencyRecord, error)
	Update(ctx context.Context, record *idempotency.IdempotencyRecord) error
}

type Service struct {
	store IdempotencyStore
	ttl   time.Duration
}

func NewService(store IdempotencyStore, ttl time.Duration) *Service {
	return &Service{
		store: store,
		ttl:   ttl,
	}
}

func (s *Service) Process(ctx context.Context, key string, requestBody []byte, handler func() (interface{}, error)) (interface{}, error) {
	if key == "" {
		return nil, errors.New(errors.CodeInvalidRequest, "idempotency key is required")
	}

	record, err := s.store.Get(ctx, key)
	if err != nil {
		if !errors.IsCode(err, errors.CodeIdempotencyNotFound) {
			return nil, err
		}

		newRecord, err := idempotency.NewIdempotencyRecord(key, requestBody, s.ttl)
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeValidationFailed, "create idempotency record")
		}

		if err := s.store.Create(ctx, newRecord); err != nil {
			return nil, errors.Wrap(err, errors.CodeDatabaseError, "store idempotency record")
		}

		result, err := handler()
		if err != nil {
			newRecord.Fail(err)
			s.store.Update(ctx, newRecord)
			return nil, err
		}

		responseBody, err := json.Marshal(result)
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternalError, "marshal response")
		}
		newRecord.Complete(responseBody)
		if err := s.store.Update(ctx, newRecord); err != nil {
			return nil, errors.Wrap(err, errors.CodeDatabaseError, "update idempotency record")
		}
		return result, nil
	}

	if record.IsExpired() {
		return nil, errors.New(errors.CodeIdempotencyNotFound, "idempotency key expired")
	}

	if err := record.CheckConflict(requestBody); err != nil {
		if errors.Is(err, idempotency.ErrProcessing) {
			return nil, errors.New(errors.CodeIdempotencyProcessing, "request is still processing")
		}
		return nil, errors.New(errors.CodeIdempotencyConflict, "idempotency key conflict: different payload")
	}

	switch record.Status {
	case idempotency.StatusSucceeded:
		var result interface{}
		if err := json.Unmarshal(record.Response, &result); err != nil {
			return nil, errors.Wrap(err, errors.CodeInternalError, "unmarshal cached response")
		}
		return result, nil
	case idempotency.StatusFailed:
		return nil, errors.New(errors.CodeInternalError, record.Error)
	default:
		return nil, errors.New(errors.CodeIdempotencyProcessing, "request is still processing")
	}
}