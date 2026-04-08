package normalize

import (
	"reflect"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

func TestRecordName(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":      "@",
		"@":     "@",
		"WWW.":  "www",
		" api ": "api",
	}

	for input, want := range tests {
		if got := RecordName(input); got != want {
			t.Fatalf("RecordName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeNameservers(t *testing.T) {
	t.Parallel()

	got := NormalizeNameservers([]string{"NS2.EXAMPLE.NET.", "ns1.example.net", "ns1.example.net"})
	want := []string{"ns1.example.net", "ns2.example.net"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeNameservers() = %#v, want %#v", got, want)
	}
}

func TestSortRecords(t *testing.T) {
	t.Parallel()

	input := []client.DNSRecord{
		{Data: "b.example.net", Priority: 20},
		{Data: "a.example.net", Priority: 10},
	}

	got := SortRecords(input)
	if got[0].Data != "a.example.net" {
		t.Fatalf("SortRecords()[0].Data = %q, want %q", got[0].Data, "a.example.net")
	}
}
