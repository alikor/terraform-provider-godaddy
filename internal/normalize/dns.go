package normalize

import (
	"cmp"
	"slices"
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

var apexNames = map[string]struct{}{
	"":  {},
	"@": {},
}

func RecordName(name string) string {
	trimmed := strings.TrimSpace(strings.ToLower(strings.TrimSuffix(name, ".")))
	if _, ok := apexNames[trimmed]; ok {
		return "@"
	}
	return trimmed
}

func RecordType(recordType string) string {
	return strings.ToUpper(strings.TrimSpace(recordType))
}

func SortRecords(records []client.DNSRecord) []client.DNSRecord {
	cloned := slices.Clone(records)
	slices.SortFunc(cloned, func(a, b client.DNSRecord) int {
		return cmp.Or(
			cmp.Compare(a.Data, b.Data),
			cmp.Compare(a.Priority, b.Priority),
			cmp.Compare(a.Weight, b.Weight),
			cmp.Compare(a.Port, b.Port),
			cmp.Compare(a.Protocol, b.Protocol),
			cmp.Compare(a.Service, b.Service),
			cmp.Compare(a.TTL, b.TTL),
		)
	})
	return cloned
}

func NormalizeNameservers(values []string) []string {
	clean := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		clean = append(clean, normalized)
	}

	slices.Sort(clean)
	return clean
}
