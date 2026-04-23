package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrBadRequest        = errors.New("bad request")
	ErrInternalServer    = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout           = errors.New("operation timeout")
	ErrInvalidInput      = errors.New("invalid input")
	ErrVoiceRecognize    = errors.New("voice recognition failed")
	ErrVoiceSynthesize   = errors.New("voice synthesis failed")
	ErrLLMRequest        = errors.New("LLM request failed")
	ErrWebSocket         = errors.New("WebSocket error")
	ErrDatabase          = errors.New("database error")
	ErrCache             = errors.New("cache error")
)

type AppError struct {
	Code    string
	Message string
	Err     error
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

func (e *AppError) Is(target error) bool {
	return e.Err == target
}

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(err error, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func Wrapf(err error, code, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), Err: err}
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

func IsInternalServer(err error) bool {
	return errors.Is(err, ErrInternalServer)
}
