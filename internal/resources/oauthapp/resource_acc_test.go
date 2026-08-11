package oauthapp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccOAuthApplication_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-oauth")
	config := func(name string) string {
		return fmt.Sprintf(`
resource "clerk_oauth_application" "test" {
  name         = %q
  callback_url = "https://app.example.com/oauth/callback"
  scopes       = "profile email"
}
`, name)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "id"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "name", name),
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "client_id"),
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "client_secret"),
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "authorize_url"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "public", "false"),
				),
			},
			{
				Config: config(name + "-renamed"),
				Check:  resource.TestCheckResourceAttr("clerk_oauth_application.test", "name", name+"-renamed"),
			},
			{
				ResourceName:            "clerk_oauth_application.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret", "scopes"},
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
		if rs.Type != "clerk_oauth_application" {
			continue
		}
		_, err := c.OAuthApps.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("OAuth application %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}
