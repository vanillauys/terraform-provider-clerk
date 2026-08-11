package instanceorgsettings_test

import (
	"context"
	"os"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkinstancesettings "github.com/clerk/clerk-sdk-go/v2/instancesettings"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccInstanceOrganizationSettings_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	acctest.PreCheck(t)
	c, err := acctest.ClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// Snapshot through the empty-PATCH read and restore after the test.
	prior, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		modes := prior.DomainsEnrollmentModes
		_, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{
			Enabled:                clerk.Bool(prior.Enabled),
			MaxAllowedMemberships:  clerk.Int64(prior.MaxAllowedMemberships),
			AdminDeleteEnabled:     clerk.Bool(prior.AdminDeleteEnabled),
			DomainsEnabled:         clerk.Bool(prior.DomainsEnabled),
			DomainsEnrollmentModes: &modes,
		})
		if err != nil {
			t.Errorf("restoring instance organization settings: %v", err)
		}
	})

	config := func(maxMemberships string) string {
		return `
resource "clerk_instance_organization_settings" "test" {
  enabled                 = true
  max_allowed_memberships = ` + maxMemberships + `
}
`
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_instance_organization_settings.test", "id"),
					resource.TestCheckResourceAttr("clerk_instance_organization_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("clerk_instance_organization_settings.test", "max_allowed_memberships", "5"),
					resource.TestCheckResourceAttrSet("clerk_instance_organization_settings.test", "creator_role"),
				),
			},
			{
				Config: config("3"),
				Check:  resource.TestCheckResourceAttr("clerk_instance_organization_settings.test", "max_allowed_memberships", "3"),
			},
			{
				ResourceName:      "clerk_instance_organization_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
