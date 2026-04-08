package provider

import (
	"fmt"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainDataSource_basic(t *testing.T) {
	domain := acctest.DiscoverTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDataSourceConfig(domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.godaddy_domain.test", "domain", domain),
					resource.TestCheckResourceAttrSet("data.godaddy_domain.test", "domain_id"),
					resource.TestCheckResourceAttrSet("data.godaddy_domain.test", "status"),
				),
			},
		},
	})
}

func testAccDomainDataSourceConfig(domain string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
data "godaddy_domain" "test" {
  domain = %q
}
`, domain)
}
