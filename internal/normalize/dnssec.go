package normalize

import (
	"cmp"
	"slices"
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

func SortDNSSECRecords(records []client.DNSSECRecord) []client.DNSSECRecord {
	cloned := slices.Clone(records)
	for i := range cloned {
		cloned[i].Algorithm = strings.ToUpper(strings.TrimSpace(cloned[i].Algorithm))
		cloned[i].DigestType = strings.ToUpper(strings.TrimSpace(cloned[i].DigestType))
		cloned[i].Flags = strings.ToUpper(strings.TrimSpace(cloned[i].Flags))
		cloned[i].Digest = strings.TrimSpace(cloned[i].Digest)
		cloned[i].PublicKey = strings.TrimSpace(cloned[i].PublicKey)
	}

	slices.SortFunc(cloned, func(a, b client.DNSSECRecord) int {
		return cmp.Or(
			cmp.Compare(a.KeyTag, b.KeyTag),
			cmp.Compare(a.Algorithm, b.Algorithm),
			cmp.Compare(a.DigestType, b.DigestType),
			cmp.Compare(a.Digest, b.Digest),
			cmp.Compare(a.Flags, b.Flags),
			cmp.Compare(a.PublicKey, b.PublicKey),
			cmp.Compare(a.MaxSignatureLife, b.MaxSignatureLife),
		)
	})

	return cloned
}
