package oauthapp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccOAuthApplication_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-oauth")
	config := func(name, body string) string {
		return fmt.Sprintf(`
resource "clerk_oauth_application" "test" {
  name         = %q
  callback_url = "https://app.example.com/oauth/callback"
  %s
}
`, name, body)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name, `scopes = "profile email"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "id"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "name", name),
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "client_id"),
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "client_secret"),
					resource.TestCheckResourceAttrSet("clerk_oauth_application.test", "authorize_url"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "public", "false"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "consent_screen_enabled", "true"),
				),
			},
			{
				Config: config(name+"-renamed", `scopes = "profile email"`),
				Check:  resource.TestCheckResourceAttr("clerk_oauth_application.test", "name", name+"-renamed"),
			},
			{
				ResourceName:            "clerk_oauth_application.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret", "scopes"},
			},
			{
				Config: config(name+"-renamed", "scopes = \"profile\"\n  public = true\n  consent_screen_enabled = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "public", "true"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "consent_screen_enabled", "false"),
				),
			},
			{
				// Remove both optional attributes: each one reverts to its default.
				// public is create-only, so this step replaces the application.
				Config: config(name+"-renamed", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "public", "false"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "consent_screen_enabled", "true"),
					resource.TestCheckResourceAttr("clerk_oauth_application.test", "scopes", "email profile"),
					checkDefaults,
				),
			},
		},
	})
}

// checkDefaults reads the application from the API: the removed attributes
// must hold the defaults on the server, not only in the state.
func checkDefaults(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	app, err := c.OAuthApps.Get(context.Background(), s.RootModule().Resources["clerk_oauth_application.test"].Primary.ID)
	if err != nil {
		return err
	}
	if app.Public || !app.ConsentScreenEnabled {
		return fmt.Errorf("application holds public=%t consent=%t, want false and true", app.Public, app.ConsentScreenEnabled)
	}
	// Clerk adds offline_access and sorts the scopes.
	if app.Scopes != "email offline_access profile" {
		return fmt.Errorf("application holds scopes %q, want the Clerk default", app.Scopes)
	}
	return nil
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
