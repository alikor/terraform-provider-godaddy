package terratest

import (
	"testing"

	"github.com/alikor/terraform-provider-godaddy/test/terratest/helpers"
)

func TestProviderSmokePrereqs(t *testing.T) {
	helpers.RequireEnv(t,
		"TF_ACC",
		"GODADDY_API_KEY",
		"GODADDY_API_SECRET",
		"GODADDY_TEST_DOMAIN",
	)
}
