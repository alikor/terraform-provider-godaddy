package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func RepoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("unable to resolve repo root: %v", err)
	}
	return root
}

func BuildLocalProvider(t *testing.T) (string, string) {
	t.Helper()

	root := RepoRoot(t)
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "terraform-provider-godaddy")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build provider: %v\n%s", err, string(output))
	}

	cliConfig := filepath.Join(tmpDir, ".terraformrc")
	content := []byte("provider_installation {\n  dev_overrides {\n    \"alikor/godaddy\" = \"" + tmpDir + "\"\n  }\n  direct {}\n}\n")
	if err := os.WriteFile(cliConfig, content, 0o600); err != nil {
		t.Fatalf("failed to write terraform CLI config: %v", err)
	}

	return tmpDir, cliConfig
}
