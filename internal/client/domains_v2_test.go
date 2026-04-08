package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGetDomainV2IncludesAndPartial(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/customers/customer-123/domains/example.com" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		includes := strings.Split(r.URL.Query().Get("includes"), ",")
		slices.Sort(includes)
		wantIncludes := []string{"actions", "authCode", "dnssecRecords"}
		if !slices.Equal(includes, wantIncludes) {
			t.Fatalf("includes = %#v, want %#v", includes, wantIncludes)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNonAuthoritativeInfo)
		if err := json.NewEncoder(w).Encode(Domain{
			Domain:        "example.com",
			DomainID:      99,
			AuthCode:      "secret",
			Actions:       []DomainAction{{Type: "DNSSEC_UPDATE", Status: "SUCCESS"}},
			DNSSECRecords: []DNSSECRecord{{KeyTag: 123, Algorithm: "RSASHA256"}},
		}); err != nil {
			t.Fatalf("unable to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := New(Config{
		APIKey:         "key",
		APISecret:      "secret",
		BaseURL:        server.URL,
		RequestTimeout: time.Second,
		PollInterval:   10 * time.Millisecond,
		MaxRetries:     0,
		RateLimitRPM:   60,
	})

	got, partial, err := c.GetDomainV2(context.Background(), "customer-123", "example.com", []string{"authCode", "actions", "dnssecRecords"})
	if err != nil {
		t.Fatalf("GetDomainV2() returned error: %v", err)
	}
	if !partial {
		t.Fatalf("partial = false, want true")
	}
	if got.DomainID != 99 {
		t.Fatalf("DomainID = %d, want 99", got.DomainID)
	}
	if got.AuthCode != "secret" {
		t.Fatalf("AuthCode = %q, want secret", got.AuthCode)
	}
	if len(got.Actions) != 1 || got.Actions[0].Type != "DNSSEC_UPDATE" {
		t.Fatalf("Actions = %#v, want DNSSEC_UPDATE", got.Actions)
	}
	if len(got.DNSSECRecords) != 1 || got.DNSSECRecords[0].KeyTag != 123 {
		t.Fatalf("DNSSECRecords = %#v, want key_tag 123", got.DNSSECRecords)
	}
}
