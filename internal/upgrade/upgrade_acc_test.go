// Package upgrade_test proves the upgrade promise of the provider: a state
// that the previous release wrote plans no change under this build.
package upgrade_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

// previousRelease is the release that writes the state. Move it to the new
// version after each minor release.
const previousRelease = "0.7.0"

// TestAccUpgradeFromPrevious applies one config with every resource that has
// a server-side read, with the previous release from the registry. Then it
// plans the same config with this build and expects an empty plan. The
// organization settings singleton is in the config because its
// domains_enrollment_modes changed from a list to a set; acctest.
// EnableOrganizations restores it afterwards. The instance settings, the
// restrictions, and the webhook stay out: the test cannot restore them.
func TestAccUpgradeFromPrevious(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	// The released provider is on the Terraform Registry only. The OpenTofu
	// run sets TF_ACC_PROVIDER_HOST to registry.opentofu.org.
	if os.Getenv("TF_ACC_PROVIDER_HOST") != "" {
		t.Skip("vanillauys/clerk is not on the OpenTofu registry")
	}
	acctest.PreCheck(t)
	acctest.EnableOrganizations(t)
	user := acctest.TestUser(t)
	name := acctest.RandomName("tf-acc-up")
	// Keys and domain labels allow no dashes; derive a clean suffix.
	suffix := strings.ToLower(strings.ReplaceAll(name, "-", ""))

	config := fmt.Sprintf(`
resource "clerk_jwt_template" "test" {
  name     = %[1]q
  lifetime = 120
  claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })
}

resource "clerk_redirect_url" "test" {
  url = "https://%[2]s.example.com/callback"
}

resource "clerk_allowlist_identifier" "test" {
  identifier = "%[2]s-allow@example.com"
}

resource "clerk_blocklist_identifier" "test" {
  identifier = "%[2]s-block@example.com"
}

resource "clerk_domain" "test" {
  name      = "%[2]s.example.com"
  proxy_url = "https://%[2]s.example.com/__clerk"
}

resource "clerk_machine" "test" {
  name = %[1]q
}

resource "clerk_oauth_application" "test" {
  name         = %[1]q
  callback_url = "https://%[2]s.example.com/oauth/callback"
  scopes       = "profile email"
}

resource "clerk_organization" "test" {
  name                    = %[1]q
  max_allowed_memberships = 5
  public_metadata = jsonencode({
    env = "acc"
  })
}

resource "clerk_organization_permission" "test" {
  name        = "Upgrade %[2]s"
  key         = "org:%[2]s:read"
  description = "upgrade"
}

resource "clerk_organization_role" "test" {
  name        = "Upgrade %[2]s"
  key         = "org:%[2]s"
  description = "upgrade"
  permissions = [clerk_organization_permission.test.id]
}

resource "clerk_organization_domain" "test" {
  organization_id = clerk_organization.test.id
  name            = "%[2]s-org.example.com"
  enrollment_mode = "manual_invitation"
}

resource "clerk_organization_membership" "test" {
  organization_id = clerk_organization.test.id
  user_id         = %[3]q
  role            = clerk_organization_role.test.key
}

resource "clerk_api_key" "test" {
  name    = %[1]q
  subject = %[3]q
  scopes  = ["read"]
}

resource "clerk_instance_organization_settings" "test" {
  enabled                  = true
  domains_enabled          = true
  domains_enrollment_modes = ["automatic_invitation", "automatic_suggestion", "manual_invitation"]
}

resource "clerk_saml_connection" "test" {
  name            = %[1]q
  domain          = "%[2]s-sso.example.com"
  provider_key    = "saml_custom"
  idp_entity_id   = "https://idp.example.com/entity"
  idp_sso_url     = "https://idp.example.com/sso"
  organization_id = clerk_organization.test.id
}
`, name, suffix, user)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"clerk": {Source: "vanillauys/clerk", VersionConstraint: previousRelease},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: acctest.ProviderFactories(),
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
