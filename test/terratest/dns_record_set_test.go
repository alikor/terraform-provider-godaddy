package terratest

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/test/terratest/helpers"
)

func TestDNSRecordSetPlan(t *testing.T) {
	helpers.RequireEnv(t,
		"TF_ACC",
		"GODADDY_API_KEY",
		"GODADDY_API_SECRET",
		"GODADDY_TEST_DOMAIN",
	)

	_, cliConfig := helpers.BuildLocalProvider(t)
	fixtureDir := filepath.Join(helpers.RepoRoot(t), "test", "terratest", "fixtures", "dns_record_set")
	env := append(os.Environ(),
		"TF_CLI_CONFIG_FILE="+cliConfig,
		"TF_VAR_godaddy_api_key="+os.Getenv("GODADDY_API_KEY"),
		"TF_VAR_godaddy_api_secret="+os.Getenv("GODADDY_API_SECRET"),
		"TF_VAR_godaddy_endpoint="+os.Getenv("GODADDY_ENDPOINT"),
		"TF_VAR_domain="+os.Getenv("GODADDY_TEST_DOMAIN"),
	)

	runTerraform(t, fixtureDir, env, "version")
	runTerraform(t, fixtureDir, env, "plan", "-input=false", "-lock=false")
}

func runTerraform(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()

	cmd := exec.Command("terraform", args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("terraform %v failed: %v\n%s", args, err, string(output))
	}
}
