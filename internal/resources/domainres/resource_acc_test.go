package domainres_test

import (
	"context"
	"fmt"
	"testing"

	sdkdomain "github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccDomain_basic(t *testing.T) {
	name := fmt.Sprintf("%s.example.com", acctest.RandomName("tf-acc"))
	// An empty proxyPath drops proxy_url from the config.
	config := func(proxyPath string) string {
		proxy := ""
		if proxyPath != "" {
			proxy = fmt.Sprintf("proxy_url = %q", "https://"+name+"/"+proxyPath)
		}
		return fmt.Sprintf(`
resource "clerk_domain" "test" {
  name = %q
  %s
}
`, name, proxy)
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
			{
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("clerk_domain.test", "proxy_url"),
					checkProxyCleared,
				),
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

// checkProxyCleared reads the domain from the API: a removed proxy_url must
// be gone on the server, not only in the state.
func checkProxyCleared(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	list, err := c.Domains.List(context.Background(), &sdkdomain.ListParams{})
	if err != nil {
		return err
	}
	id := s.RootModule().Resources["clerk_domain.test"].Primary.ID
	for _, d := range list.Domains {
		if d.ID == id && d.ProxyURL != nil && *d.ProxyURL != "" {
			return fmt.Errorf("proxy_url is %q on the server, want empty", *d.ProxyURL)
		}
	}
	return nil
}
