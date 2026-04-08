package client

import "testing"

func TestNewEndpointRateLimiterClampsRPM(t *testing.T) {
	t.Parallel()

	low := newEndpointRateLimiter(0)
	if low.rpm != 1 {
		t.Fatalf("low.rpm = %d, want 1", low.rpm)
	}

	high := newEndpointRateLimiter(99)
	if high.rpm != 60 {
		t.Fatalf("high.rpm = %d, want 60", high.rpm)
	}
}
