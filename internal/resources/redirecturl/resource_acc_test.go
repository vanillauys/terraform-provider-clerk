package redirecturl_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccRedirectURL_basic(t *testing.T) {
	url := fmt.Sprintf("https://%s.example.com/sso-callback", acctest.RandomName("tf-acc"))
	config := fmt.Sprintf(`
resource "clerk_redirect_url" "test" {
  url = %q
}
`, url)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_redirect_url.test", "id"),
					resource.TestCheckResourceAttr("clerk_redirect_url.test", "url", url),
				),
			},
			{
				ResourceName:      "clerk_redirect_url.test",
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
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clerk_redirect_url" {
			continue
		}
		_, err := c.RedirectURLs.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("redirect URL %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}
