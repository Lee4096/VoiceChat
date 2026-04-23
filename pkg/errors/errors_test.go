package errors

import (
	"errors"
	"testing"
)

func TestAppError(t *testing.T) {
	err := New("TEST_ERROR", "test message")
	if err.Error() != "TEST_ERROR: test message" {
		t.Errorf("Error() = %s, want TEST_ERROR: test message", err.Error())
	}
}

func TestAppErrorWithWrapped(t *testing.T) {
	inner := errors.New("inner error")
	err := Wrap(inner, "OUTER", "outer message")
	if err.Error() != "OUTER: outer message: inner error" {
		t.Errorf("Error() = %s, want OUTER: outer message: inner error", err.Error())
	}
}

func TestAppErrorWrapf(t *testing.T) {
	err := Wrapf(nil, "TEST", "formatted %s", "message")
	if err.Code != "TEST" {
		t.Errorf("Code = %s, want TEST", err.Code)
	}
	if err.Message != "formatted message" {
		t.Errorf("Message = %s, want formatted message", err.Message)
	}
}

func TestIsErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"NotFound", ErrNotFound, ErrNotFound, true},
		{"Unauthorized", ErrUnauthorized, ErrUnauthorized, true},
		{"WrappedNotFound", Wrap(errors.New("inner"), "CODE", "msg"), ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Error("IsNotFound(ErrNotFound) = false, want true")
	}

	if IsNotFound(errors.New("other")) {
		t.Error("IsNotFound(other) = true, want false")
	}
}
