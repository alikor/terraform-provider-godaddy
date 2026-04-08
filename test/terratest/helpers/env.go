package helpers

import (
	"os"
	"testing"
)

func RequireEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if os.Getenv(key) == "" {
			t.Skipf("skipping Terratest because %s is not set", key)
		}
	}
}
