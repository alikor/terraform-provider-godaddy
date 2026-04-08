package provider

import (
	"context"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

func TestForwardMaskRoundTrip(t *testing.T) {
	t.Parallel()

	mask := &client.ForwardMask{
		Title:       "Example",
		Description: "Forwarded page",
		Keywords:    "one,two",
	}

	obj := forwardMaskObjectFromAPI(mask)
	got, err := forwardMaskFromObject(context.Background(), obj)
	if err != nil {
		t.Fatalf("forwardMaskFromObject() returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil mask")
	}
	if *got != *mask {
		t.Fatalf("round-trip mismatch: got %#v want %#v", got, mask)
	}
}
