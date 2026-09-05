package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	apperrors "arca-invoice-proxy/internal/domain/errors"
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
		return nil, apperrors.New(apperrors.CodeInvalidRequest, "idempotency key is required")
	}

	record, err := s.store.Get(ctx, key)
	if err != nil {
		if !apperrors.IsCode(err, apperrors.CodeIdempotencyNotFound) {
			return nil, err
		}

		newRecord, err := idempotency.NewIdempotencyRecord(key, requestBody, s.ttl)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeValidationFailed, "create idempotency record")
		}

		if err := s.store.Create(ctx, newRecord); err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "store idempotency record")
		}

		result, err := handler()
		if err != nil {
			newRecord.Fail(err)
			s.store.Update(ctx, newRecord)
			return nil, err
		}

		responseBody, err := json.Marshal(result)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeInternalError, "marshal response")
		}
		newRecord.Complete(responseBody)
		if err := s.store.Update(ctx, newRecord); err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "update idempotency record")
		}
		return result, nil
	}

	if record.IsExpired() {
		return nil, apperrors.New(apperrors.CodeIdempotencyNotFound, "idempotency key expired")
	}

	if err := record.CheckConflict(requestBody); err != nil {
		if errors.Is(err, idempotency.ErrProcessing) {
			return nil, apperrors.New(apperrors.CodeIdempotencyProcessing, "request is still processing")
		}
		return nil, apperrors.New(apperrors.CodeIdempotencyConflict, "idempotency key conflict: different payload")
	}

	switch record.Status {
	case idempotency.StatusSucceeded:
		var result interface{}
		if err := json.Unmarshal(record.Response, &result); err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeInternalError, "unmarshal cached response")
		}
		return result, nil
	case idempotency.StatusFailed:
		return nil, apperrors.New(apperrors.CodeInternalError, record.Error)
	default:
		return nil, apperrors.New(apperrors.CodeIdempotencyProcessing, "request is still processing")
	}
}
