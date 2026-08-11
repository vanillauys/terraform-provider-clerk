package domainres_test

import (
	"context"
	"fmt"
	"testing"

	sdkdomain "github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccDomain_basic(t *testing.T) {
	name := fmt.Sprintf("%s.example.com", acctest.RandomName("tf-acc"))
	config := func(proxyPath string) string {
		return fmt.Sprintf(`
resource "clerk_domain" "test" {
  name      = %q
  proxy_url = "https://%s/%s"
}
`, name, name, proxyPath)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config("__clerk"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_domain.test", "id"),
					resource.TestCheckResourceAttr("clerk_domain.test", "name", name),
					resource.TestCheckResourceAttr("clerk_domain.test", "is_satellite", "true"),
					resource.TestCheckResourceAttrSet("clerk_domain.test", "frontend_api_url"),
				),
			},
			{
				Config: config("__proxy"),
				Check:  resource.TestCheckResourceAttr("clerk_domain.test", "proxy_url", "https://"+name+"/__proxy"),
			},
			{
				ResourceName:      "clerk_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func checkDestroy(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	list, err := c.Domains.List(context.Background(), &sdkdomain.ListParams{})
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clerk_domain" {
			continue
		}
		for _, d := range list.Domains {
			if d.ID == rs.Primary.ID {
				return fmt.Errorf("domain %s still exists", rs.Primary.ID)
			}
		}
	}
	return nil
}
