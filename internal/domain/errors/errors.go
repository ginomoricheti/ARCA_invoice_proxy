package errors

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeInvalidRequest         ErrorCode = "invalid_request"
	CodeInvalidCUIT            ErrorCode = "invalid_cuit"
	CodeInvalidInvoiceType     ErrorCode = "invalid_invoice_type"
	CodeInvalidAmount          ErrorCode = "invalid_amount"
	CodeInvalidAPIKey          ErrorCode = "invalid_api_key"
	CodeAPIKeyRevoked          ErrorCode = "api_key_revoked"
	CodeAPIKeyExpired          ErrorCode = "api_key_expired"
	CodeIdempotencyConflict    ErrorCode = "idempotency_conflict"
	CodeIdempotencyProcessing  ErrorCode = "idempotency_processing"
	CodeIdempotencyNotFound    ErrorCode = "idempotency_not_found"
	CodeARCAAuthFailed         ErrorCode = "arca_authentication_failed"
	CodeARCARejected           ErrorCode = "arca_rejected"
	CodeARCAServiceUnavailable ErrorCode = "arca_service_unavailable"
	CodeDatabaseError          ErrorCode = "database_error"
	CodeInternalError          ErrorCode = "internal_error"
	CodeNotFound               ErrorCode = "not_found"
	CodeUnauthorized           ErrorCode = "unauthorized"
	CodeForbidden              ErrorCode = "forbidden"
	CodeValidationFailed       ErrorCode = "validation_failed"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
	Details map[string]string
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: make(map[string]string),
	}
}

func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Details: make(map[string]string),
	}
}

func (e *AppError) WithDetail(key, value string) *AppError {
	e.Details[key] = value
	return e
}

func IsCode(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

func GetCode(err error) ErrorCode {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternalError
}

func IsNotFound(err error) bool {
	return IsCode(err, CodeNotFound)
}

func IsUnauthorized(err error) bool {
	return IsCode(err, CodeUnauthorized) || IsCode(err, CodeInvalidAPIKey) || IsCode(err, CodeAPIKeyRevoked) || IsCode(err, CodeAPIKeyExpired)
}

func IsForbidden(err error) bool {
	return IsCode(err, CodeForbidden)
}

func IsValidation(err error) bool {
	return IsCode(err, CodeValidationFailed) || IsCode(err, CodeInvalidRequest) || IsCode(err, CodeInvalidCUIT) || IsCode(err, CodeInvalidInvoiceType) || IsCode(err, CodeInvalidAmount)
}

func IsIdempotency(err error) bool {
	return IsCode(err, CodeIdempotencyConflict) || IsCode(err, CodeIdempotencyProcessing) || IsCode(err, CodeIdempotencyNotFound)
}

func IsARCAError(err error) bool {
	return IsCode(err, CodeARCAAuthFailed) || IsCode(err, CodeARCARejected) || IsCode(err, CodeARCAServiceUnavailable)
}

func IsDatabase(err error) bool {
	return IsCode(err, CodeDatabaseError)
}
