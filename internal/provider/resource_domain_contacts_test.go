package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestContactObjectRoundTrip(t *testing.T) {
	t.Parallel()

	original := client.Contact{
		NameFirst:    "Jane",
		NameLast:     "Doe",
		Email:        "jane@example.com",
		Phone:        "+1.4805550100",
		Organization: "Example Inc",
		AddressMailing: client.MailingAddress{
			Address1:   "123 Main St",
			City:       "Tempe",
			State:      "AZ",
			PostalCode: "85281",
			Country:    "us",
		},
	}

	obj := contactObjectFromAPI(original)
	got, err := contactFromObject(context.Background(), obj)
	if err != nil {
		t.Fatalf("contactFromObject() returned error: %v", err)
	}

	if got.AddressMailing.Country != "US" {
		t.Fatalf("country = %q, want US", got.AddressMailing.Country)
	}
	if got.NameFirst != original.NameFirst || got.NameLast != original.NameLast || got.Email != original.Email {
		t.Fatalf("round-trip mismatch: got %#v want %#v", got, original)
	}
}

func TestUseV2ContactsUpdate(t *testing.T) {
	t.Parallel()

	if useV2ContactsUpdate(types.StringNull()) {
		t.Fatalf("null identity document id should not use v2")
	}
	if useV2ContactsUpdate(types.StringValue("")) {
		t.Fatalf("empty identity document id should not use v2")
	}
	if !useV2ContactsUpdate(types.StringValue("doc-123")) {
		t.Fatalf("non-empty identity document id should use v2")
	}
}

func TestPatchDomainContactsV2(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/v2/customers/customer-123/domains/example.com/contacts" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Request-Id"); got == "" {
			t.Fatalf("expected X-Request-Id header to be set")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unable to read body: %v", err)
		}

		var payload client.DomainContactsV2Update
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unable to decode payload: %v", err)
		}

		if payload.IdentityDocumentID != "doc-123" {
			t.Fatalf("identityDocumentId = %q, want doc-123", payload.IdentityDocumentID)
		}
		if payload.Registrant.Email != "jane@example.com" {
			t.Fatalf("registrant email = %q, want jane@example.com", payload.Registrant.Email)
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c := client.New(client.Config{
		APIKey:         "key",
		APISecret:      "secret",
		BaseURL:        server.URL,
		RequestTimeout: time.Second,
		PollInterval:   10 * time.Millisecond,
		MaxRetries:     0,
		RateLimitRPM:   60,
	})

	err := c.PatchDomainContactsV2(context.Background(), "customer-123", "example.com", client.DomainContactsV2Update{
		Registrant: client.Contact{
			Email: "jane@example.com",
		},
		IdentityDocumentID: "doc-123",
	})
	if err != nil {
		t.Fatalf("PatchDomainContactsV2() returned error: %v", err)
	}
}
