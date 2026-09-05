package tests

import (
	"testing"
	"time"

	"arca-invoice-proxy/internal/domain/idempotency"
)

func TestIdempotencyRecord_Creation(t *testing.T) {
	requestBody := []byte(`{"key": "value"}`)
	ttl := 24 * time.Hour

	record, err := idempotency.NewIdempotencyRecord("test-key", requestBody, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if record.Key != "test-key" {
		t.Error("key mismatch")
	}
	if record.RequestHash == "" {
		t.Error("request hash should be set")
	}
	if record.Status != idempotency.StatusProcessing {
		t.Error("initial status should be processing")
	}
	if record.Response != nil {
		t.Error("response should be nil initially")
	}
	if record.Error != "" {
		t.Error("error should be empty initially")
	}
	if record.CreatedAt.IsZero() {
		t.Error("created at should be set")
	}
	if record.UpdatedAt.IsZero() {
		t.Error("updated at should be set")
	}
	if record.ExpiresAt.IsZero() {
		t.Error("expires at should be set")
	}
}

func TestIdempotencyRecord_Creation_Errors(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		requestBody []byte
		hasError    bool
	}{
		{"empty key", "", []byte(`{"a": 1}`), true},
		{"empty body", "key", []byte{}, true},
		{"valid", "key", []byte(`{"a": 1}`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := idempotency.NewIdempotencyRecord(tt.key, tt.requestBody, time.Hour)
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

func TestIdempotencyRecord_CheckConflict(t *testing.T) {
	requestBody := []byte(`{"key": "value"}`)
	record, _ := idempotency.NewIdempotencyRecord("test-key", requestBody, time.Hour)

	// Same payload should not conflict
	err := record.CheckConflict(requestBody)
	if err != nil {
		t.Errorf("same payload should not conflict: %v", err)
	}

	// Different payload should conflict
	differentBody := []byte(`{"key": "different"}`)
	err = record.CheckConflict(differentBody)
	if err == nil {
		t.Error("different payload should conflict")
	}
	if err != idempotency.ErrIdempotencyConflict {
		t.Errorf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestIdempotencyRecord_Complete(t *testing.T) {
	record, _ := idempotency.NewIdempotencyRecord("test-key", []byte(`{"a": 1}`), time.Hour)

	response := []byte(`{"result": "ok"}`)
	record.Complete(response)

	if record.Status != idempotency.StatusSucceeded {
		t.Error("status should be succeeded")
	}
	if string(record.Response) != string(response) {
		t.Error("response should be stored")
	}
	if record.Error != "" {
		t.Error("error should be empty")
	}
	if record.UpdatedAt.IsZero() {
		t.Error("updated at should be updated")
	}
}

func TestIdempotencyRecord_Fail(t *testing.T) {
	record, _ := idempotency.NewIdempotencyRecord("test-key", []byte(`{"a": 1}`), time.Hour)

	testErr := &testError{msg: "something failed"}
	record.Fail(testErr)

	if record.Status != idempotency.StatusFailed {
		t.Error("status should be failed")
	}
	if record.Response != nil {
		t.Error("response should be nil")
	}
	if record.Error != "something failed" {
		t.Errorf("error should be stored, got %q", record.Error)
	}
	if record.UpdatedAt.IsZero() {
		t.Error("updated at should be updated")
	}
}

func TestIdempotencyRecord_IsExpired(t *testing.T) {
	record, _ := idempotency.NewIdempotencyRecord("test-key", []byte(`{"a": 1}`), time.Hour)

	if record.IsExpired() {
		t.Error("new record should not be expired")
	}

	// Create expired record
	past := time.Now().UTC().Add(-1 * time.Hour)
	expiredRecord := &idempotency.IdempotencyRecord{
		Key:       "expired",
		Status:    idempotency.StatusProcessing,
		ExpiresAt: past,
	}
	if !expiredRecord.IsExpired() {
		t.Error("past expiration should be expired")
	}
}

func TestIdempotency_Status_Parse(t *testing.T) {
	tests := []struct {
		input    string
		expected idempotency.IdempotencyStatus
		hasError bool
	}{
		{"processing", idempotency.StatusProcessing, false},
		{"succeeded", idempotency.StatusSucceeded, false},
		{"failed", idempotency.StatusFailed, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := idempotency.ParseStatus(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
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

func TestIdempotency_Status_Valid(t *testing.T) {
	tests := []struct {
		input    idempotency.IdempotencyStatus
		expected bool
	}{
		{idempotency.StatusProcessing, true},
		{idempotency.StatusSucceeded, true},
		{idempotency.StatusFailed, true},
		{idempotency.IdempotencyStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.Valid() != tt.expected {
				t.Errorf("expected Valid()=%v for %q", tt.expected, tt.input)
			}
		})
	}
}

func TestIdempotencyRecord_RequestHash(t *testing.T) {
	requestBody := []byte(`{"key": "value"}`)
	record, _ := idempotency.NewIdempotencyRecord("test-key", requestBody, time.Hour)

	// Same payload should produce same hash
	record2, _ := idempotency.NewIdempotencyRecord("test-key-2", requestBody, time.Hour)
	if record.RequestHash != record2.RequestHash {
		t.Error("same payload should produce same request hash")
	}

	// Different payload should produce different hash
	differentBody := []byte(`{"key": "different"}`)
	record3, _ := idempotency.NewIdempotencyRecord("test-key-3", differentBody, time.Hour)
	if record.RequestHash == record3.RequestHash {
		t.Error("different payload should produce different request hash")
	}

	if len(record.RequestHash) != 64 {
		t.Errorf("hash length should be 64, got %d", len(record.RequestHash))
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}