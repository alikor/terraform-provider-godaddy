package provider

import (
	"reflect"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

func TestDiffDNSSECRecords(t *testing.T) {
	t.Parallel()

	current := []client.DNSSECRecord{
		{KeyTag: 100, Algorithm: "RSASHA256", DigestType: "SHA256", Digest: "AAAA"},
		{KeyTag: 200, Algorithm: "RSASHA256", DigestType: "SHA256", Digest: "BBBB"},
	}
	desired := []client.DNSSECRecord{
		{KeyTag: 100, Algorithm: "RSASHA256", DigestType: "SHA256", Digest: "AAAA"},
		{KeyTag: 300, Algorithm: "RSASHA256", DigestType: "SHA256", Digest: "CCCC"},
	}

	toAdd, toRemove := diffDNSSECRecords(current, desired)

	wantAdd := []client.DNSSECRecord{
		{KeyTag: 300, Algorithm: "RSASHA256", DigestType: "SHA256", Digest: "CCCC"},
	}
	wantRemove := []client.DNSSECRecord{
		{KeyTag: 200, Algorithm: "RSASHA256", DigestType: "SHA256", Digest: "BBBB"},
	}

	if !reflect.DeepEqual(toAdd, wantAdd) {
		t.Fatalf("toAdd = %#v, want %#v", toAdd, wantAdd)
	}
	if !reflect.DeepEqual(toRemove, wantRemove) {
		t.Fatalf("toRemove = %#v, want %#v", toRemove, wantRemove)
	}
}
