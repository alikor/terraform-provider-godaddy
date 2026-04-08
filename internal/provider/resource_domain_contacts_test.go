package provider

import (
	"context"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
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
