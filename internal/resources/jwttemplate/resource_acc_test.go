package jwttemplate_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccJWTTemplate_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-jwt")
	config := func(body string) string {
		return fmt.Sprintf(`
resource "clerk_jwt_template" "test" {
  name = %q
  %s
}
`, name, body)
	}
	claims := `claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config("lifetime = 3600\n" + claims),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_jwt_template.test", "id"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "name", name),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "lifetime", "3600"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "signing_algorithm", "RS256"),
				),
			},
			{
				Config: config("lifetime = 7200\n" + claims),
				Check:  resource.TestCheckResourceAttr("clerk_jwt_template.test", "lifetime", "7200"),
			},
			{
				ResourceName:            "clerk_jwt_template.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"signing_key"},
			},
			{
				Config: config(`lifetime           = 7200
  allowed_clock_skew = 10
  custom_signing_key = true
  signing_algorithm  = "HS256"
  signing_key        = "tf-acc-hmac-secret-0123456789abcdef"
  ` + claims),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "allowed_clock_skew", "10"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "custom_signing_key", "true"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "signing_algorithm", "HS256"),
				),
			},
			{
				// Remove every optional attribute: each one reverts to its default.
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "lifetime", "60"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "allowed_clock_skew", "5"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "custom_signing_key", "false"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "signing_algorithm", "RS256"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "claims", "{}"),
					checkDefaults,
				),
			},
		},
	})
}

// TestAccJWTTemplate_writeOnly proves that signing_key_wo stays out of the
// state, and that signing_key_wo_version alone decides when a new value
// reaches Clerk.
func TestAccJWTTemplate_writeOnly(t *testing.T) {
	name := acctest.RandomName("tf-acc-jwt-wo")
	config := func(key string, version int) string {
		return fmt.Sprintf(`
resource "clerk_jwt_template" "test" {
  name                   = %q
  custom_signing_key     = true
  signing_algorithm      = "HS256"
  signing_key_wo         = %q
  signing_key_wo_version = %d
}
`, name, key, version)
	}
	resource.Test(t, resource.TestCase{
		// A write-only attribute needs Terraform 1.11 or later.
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config("tf-acc-hmac-secret-a-0123456789abcdef", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("clerk_jwt_template.test", "signing_key"),
					resource.TestCheckNoResourceAttr("clerk_jwt_template.test", "signing_key_wo"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "signing_key_wo_version", "1"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "custom_signing_key", "true"),
				),
			},
			{
				// A new value without a new version plans no change.
				Config: config("tf-acc-hmac-secret-b-0123456789abcdef", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				// A new version sends the value in place.
				Config: config("tf-acc-hmac-secret-b-0123456789abcdef", 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction("clerk_jwt_template.test", plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckNoResourceAttr("clerk_jwt_template.test", "signing_key_wo"),
			},
		},
	})
}

// checkDefaults reads the template from the API: the removed attributes
// must hold the defaults on the server, not only in the state.
func checkDefaults(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	tpl, err := c.JWTTemplates.Get(context.Background(), s.RootModule().Resources["clerk_jwt_template.test"].Primary.ID)
	if err != nil {
		return err
	}
	if tpl.Lifetime != 60 || tpl.AllowedClockSkew != 5 || tpl.CustomSigningKey || tpl.SigningAlgorithm != "RS256" {
		return fmt.Errorf("template holds lifetime=%d skew=%d custom=%t alg=%s, want the defaults",
			tpl.Lifetime, tpl.AllowedClockSkew, tpl.CustomSigningKey, tpl.SigningAlgorithm)
	}
	if string(tpl.Claims) != "{}" {
		return fmt.Errorf("template claims are %s, want {}", tpl.Claims)
	}
	return nil
}

func checkDestroy(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clerk_jwt_template" {
			continue
		}
		_, err := c.JWTTemplates.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("JWT template %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}
