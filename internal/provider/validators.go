package provider

import (
	"fmt"
	"strings"
)

var supportedManagedRecordTypes = map[string]struct{}{
	"A":     {},
	"AAAA":  {},
	"CNAME": {},
	"MX":    {},
	"SRV":   {},
	"TXT":   {},
}

func validateManagedRecordType(recordType string) error {
	if _, ok := supportedManagedRecordTypes[strings.ToUpper(recordType)]; ok {
		return nil
	}

	return fmt.Errorf("unsupported record type %q; supported types are A, AAAA, CNAME, MX, SRV, TXT", recordType)
}
