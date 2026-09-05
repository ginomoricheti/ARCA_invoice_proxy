package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key conflict: different payload")
	ErrKeyNotFound            = errors.New("idempotency key not found")
	ErrKeyExpired             = errors.New("idempotency key expired")
	ErrProcessing             = errors.New("request is still processing")
)

type IdempotencyStatus string

const (
	StatusProcessing IdempotencyStatus = "processing"
	StatusSucceeded  IdempotencyStatus = "succeeded"
	StatusFailed     IdempotencyStatus = "failed"
)

func ParseStatus(s string) (IdempotencyStatus, error) {
	switch s {
	case "processing":
		return StatusProcessing, nil
	case "succeeded":
		return StatusSucceeded, nil
	case "failed":
		return StatusFailed, nil
	default:
		return "", errors.New("invalid idempotency status: " + s)
	}
}

func (s IdempotencyStatus) String() string {
	return string(s)
}

func (s IdempotencyStatus) Valid() bool {
	return s == StatusProcessing || s == StatusSucceeded || s == StatusFailed
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Status      IdempotencyStatus
	Response    []byte
	Error       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
}

func NewIdempotencyRecord(key string, requestBody []byte, ttl time.Duration) (*IdempotencyRecord, error) {
	if key == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	if len(requestBody) == 0 {
		return nil, errors.New("request body is required")
	}

	requestHash := computeHash(requestBody)
	now := time.Now().UTC()

	return &IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Status:      StatusProcessing,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}, nil
}

func (r *IdempotencyRecord) CheckConflict(requestBody []byte) error {
	newHash := computeHash(requestBody)
	if r.RequestHash != newHash {
		return ErrIdempotencyConflict
	}
	if r.Status == StatusProcessing {
		return ErrProcessing
	}
	return nil
}

func (r *IdempotencyRecord) Complete(response []byte) {
	r.Status = StatusSucceeded
	r.Response = response
	r.Error = ""
	r.UpdatedAt = time.Now().UTC()
}

func (r *IdempotencyRecord) Fail(err error) {
	r.Status = StatusFailed
	r.Response = nil
	if err != nil {
		r.Error = err.Error()
	}
	r.UpdatedAt = time.Now().UTC()
}

func (r *IdempotencyRecord) IsExpired() bool {
	return time.Now().UTC().After(r.ExpiresAt)
}

func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

type IdempotencyStore interface {
	Create(ctx interface{}, record *IdempotencyRecord) error
	Get(ctx interface{}, key string) (*IdempotencyRecord, error)
	Update(ctx interface{}, record *IdempotencyRecord) error
}
