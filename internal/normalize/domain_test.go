package normalize

import "testing"

func TestDomain(t *testing.T) {
	t.Parallel()

	got, err := Domain("BÜCHER.DE.")
	if err != nil {
		t.Fatalf("Domain() returned error: %v", err)
	}
	if got != "xn--bcher-kva.de" {
		t.Fatalf("Domain() = %q, want %q", got, "xn--bcher-kva.de")
	}
}
