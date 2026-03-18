package usecase

import "testing"

func TestAppError_Error(t *testing.T) {
	var e *AppError
	if got := e.Error(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	e = &AppError{Code: "x", Message: "hello"}
	if got := e.Error(); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

