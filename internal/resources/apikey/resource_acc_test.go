package apikey_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccAPIKey_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	acctest.PreCheck(t)
	subject := acctest.TestUser(t)
	name := acctest.RandomName("tf-acc-key")
	// expiration holds the timestamp after the first apply with a lifetime.
	var expiration string

	config := func(extra string) string {
		return fmt.Sprintf(`
resource "clerk_api_key" "test" {
  name    = %q
  subject = %q
%s
}
`, name, subject, extra)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(`  scopes = ["read"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_api_key.test", "id"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "name", name),
					resource.TestCheckResourceAttr("clerk_api_key.test", "subject", subject),
					resource.TestCheckResourceAttr("clerk_api_key.test", "scopes.#", "1"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "claims", "{}"),
					resource.TestCheckResourceAttrSet("clerk_api_key.test", "secret"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "type", "api_key"),
				),
			},
			{
				Config: config(`  scopes      = ["read"]
  description = "acceptance test key"
  claims      = jsonencode({ env = "acc" })
  seconds_until_expiration = 86400`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_api_key.test", "description", "acceptance test key"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "claims", `{"env":"acc"}`),
					resource.TestCheckResourceAttrWith("clerk_api_key.test", "expiration", func(v string) error {
						expiration = v
						return nil
					}),
				),
			},
			{
				ResourceName:      "clerk_api_key.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Clerk does not return the lifetime, only the expiration.
				ImportStateVerifyIgnore: []string{"seconds_until_expiration"},
			},
			{
				// An unrelated change must not restart the lifetime count.
				Config: config(`  scopes      = ["read"]
  description = "acceptance test key, renamed"
  claims      = jsonencode({ env = "acc" })
  seconds_until_expiration = 86400`),
				Check: resource.TestCheckResourceAttrWith("clerk_api_key.test", "expiration", func(v string) error {
					if v != expiration {
						return fmt.Errorf("expiration moved from %s to %s", expiration, v)
					}
					return nil
				}),
			},
			{
				// Remove every optional attribute: description reverts to null,
				// claims and scopes revert to their defaults.
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("clerk_api_key.test", "description"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "claims", "{}"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "scopes.#", "0"),
					checkDescriptionCleared,
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
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clerk_api_key" {
			continue
		}
		_, err := c.APIKeys.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("API key %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// checkDescriptionCleared reads the key from the API: a removed description
// must be gone on the server, and the removed claims and scopes must hold
// their defaults there, not only in the state.
func checkDescriptionCleared(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	k, err := c.APIKeys.Get(context.Background(), s.RootModule().Resources["clerk_api_key.test"].Primary.ID)
	if err != nil {
		return err
	}
	if k.Description != nil && *k.Description != "" {
		return fmt.Errorf("description is %q on the server, want empty", *k.Description)
	}
	if string(k.Claims) != "{}" || len(k.Scopes) != 0 {
		return fmt.Errorf("key holds claims=%s scopes=%v, want {} and none", k.Claims, k.Scopes)
	}
	return nil
}
