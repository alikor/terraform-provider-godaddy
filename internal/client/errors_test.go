package client

import "testing"

func TestParseAPIError(t *testing.T) {
	t.Parallel()

	err := parseAPIError(422, []byte(`{"code":"INVALID_BODY","message":"bad request","fields":[{"path":"domain","message":"required"}]}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Fatalf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
	if apiErr.Code != "INVALID_BODY" {
		t.Fatalf("Code = %q, want INVALID_BODY", apiErr.Code)
	}
	if len(apiErr.Fields) != 1 || apiErr.Fields[0].Path != "domain" {
		t.Fatalf("Fields = %#v, want domain field", apiErr.Fields)
	}
}

func TestParseRateLimitError(t *testing.T) {
	t.Parallel()

	err := parseAPIError(429, []byte(`{"code":"RATE_LIMIT","message":"slow down","retryAfterSec":7}`))
	rateErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
	if rateErr.RetryAfterSec != 7 {
		t.Fatalf("RetryAfterSec = %d, want 7", rateErr.RetryAfterSec)
	}
}
