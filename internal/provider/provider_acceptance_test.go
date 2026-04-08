package provider

import (
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/acctest"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"godaddy": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	acctest.RequireEnv(t,
		acctest.EnvAPIKey,
		acctest.EnvAPISecret,
	)
}
