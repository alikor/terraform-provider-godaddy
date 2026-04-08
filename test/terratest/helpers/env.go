package helpers

import (
	"os"
	"os/exec"
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

func RequireTerraformCLI(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skipf("skipping Terratest because terraform is not installed: %v", err)
	}
}
