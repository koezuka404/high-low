package usecase

import "testing"

func TestRateLimitError_Error(t *testing.T) {
	e := &RateLimitError{RetryAfterSec: 1}
	if e.Error() != "rate limit exceeded" {
		t.Fatalf("unexpected message: %s", e.Error())
	}
}

