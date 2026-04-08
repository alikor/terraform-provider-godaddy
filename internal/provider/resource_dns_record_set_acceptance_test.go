package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/acctest"
	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDNSRecordSetResource_basic(t *testing.T) {
	domain := acctest.DiscoverTestDomain(t)
	recordName := fmt.Sprintf("tfacc-codex-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordSetResourceConfig(domain, recordName, "acceptance-one"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("godaddy_dns_record_set.test", "domain", domain),
					resource.TestCheckResourceAttr("godaddy_dns_record_set.test", "type", "TXT"),
					resource.TestCheckResourceAttr("godaddy_dns_record_set.test", "name", recordName),
					resource.TestCheckResourceAttr("godaddy_dns_record_set.test", "records.0.data", "acceptance-one"),
					resource.TestCheckResourceAttr("godaddy_dns_record_set.test", "records.0.ttl", "600"),
				),
			},
			{
				ResourceName:      "godaddy_dns_record_set.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccDNSRecordSetResourceConfig(domain, recordName, "acceptance-two"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("godaddy_dns_record_set.test", "records.0.data", "acceptance-two"),
				),
			},
		},
	})
}

func testAccDNSRecordSetResourceConfig(domain, name, value string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
resource "godaddy_dns_record_set" "test" {
  domain = %q
  type   = "TXT"
  name   = %q

  records = [
    {
      data = %q
      ttl  = 600
    }
  ]
}
`, domain, name, value)
}

func testAccCheckDNSRecordSetDestroy(state *terraform.State) error {
	c := acctest.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "godaddy_dns_record_set" {
			continue
		}

		domain := rs.Primary.Attributes["domain"]
		recordType := rs.Primary.Attributes["type"]
		name := rs.Primary.Attributes["name"]

		records, err := c.GetDNSRecordSet(ctx, domain, recordType, name)
		if err != nil {
			var apiErr *client.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				continue
			}
			return err
		}

		if len(records) > 0 {
			return fmt.Errorf("dns record set %s,%s,%s still exists", domain, recordType, name)
		}
	}

	return nil
}
