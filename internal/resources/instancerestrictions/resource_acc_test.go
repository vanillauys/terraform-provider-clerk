package instancerestrictions_test

import (
	"context"
	"os"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkinstancesettings "github.com/clerk/clerk-sdk-go/v2/instancesettings"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccInstanceRestrictions_basic(t *testing.T) {
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
	prior, err := c.InstanceSettings.UpdateRestrictions(ctx, &sdkinstancesettings.UpdateRestrictionsParams{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := c.InstanceSettings.UpdateRestrictions(ctx, &sdkinstancesettings.UpdateRestrictionsParams{
			Allowlist:                   clerk.Bool(prior.Allowlist),
			Blocklist:                   clerk.Bool(prior.Blocklist),
			BlockEmailSubaddresses:      clerk.Bool(prior.BlockEmailSubaddresses),
			BlockDisposableEmailDomains: clerk.Bool(prior.BlockDisposableEmailDomains),
			IgnoreDotsForGmailAddresses: clerk.Bool(prior.IgnoreDotsForGmailAddresses),
		})
		if err != nil {
			t.Errorf("restoring instance restrictions: %v", err)
		}
	})

	config := func(v string) string {
		return `
resource "clerk_instance_restrictions" "test" {
  block_email_subaddresses = ` + v + `
}
`
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_instance_restrictions.test", "id"),
					resource.TestCheckResourceAttr("clerk_instance_restrictions.test", "block_email_subaddresses", "true"),
					resource.TestCheckResourceAttrSet("clerk_instance_restrictions.test", "allowlist"),
					resource.TestCheckResourceAttrSet("clerk_instance_restrictions.test", "blocklist"),
				),
			},
			{
				Config: config("false"),
				Check:  resource.TestCheckResourceAttr("clerk_instance_restrictions.test", "block_email_subaddresses", "false"),
			},
			{
				ResourceName:      "clerk_instance_restrictions.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
