package normalize

import (
	"strings"

	"golang.org/x/net/idna"
)

func Domain(input string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(strings.TrimSuffix(input, ".")))
	if trimmed == "" {
		return "", ErrEmptyValue("domain")
	}

	ascii, err := idna.Lookup.ToASCII(trimmed)
	if err != nil {
		return "", err
	}

	return ascii, nil
}

func FQDN(input string) (string, error) {
	return Domain(input)
}
