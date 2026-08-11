package identifier_test

import (
	"context"
	"fmt"
	"testing"

	sdkallowlist "github.com/clerk/clerk-sdk-go/v2/allowlistidentifier"
	sdkblocklist "github.com/clerk/clerk-sdk-go/v2/blocklistidentifier"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccAllowlistIdentifier_basic(t *testing.T) {
	email := fmt.Sprintf("%s@example.com", acctest.RandomName("tf-acc"))
	config := fmt.Sprintf(`
resource "clerk_allowlist_identifier" "test" {
  identifier = %q
}
`, email)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkIdentifierDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_allowlist_identifier.test", "id"),
					resource.TestCheckResourceAttr("clerk_allowlist_identifier.test", "identifier", email),
					resource.TestCheckResourceAttr("clerk_allowlist_identifier.test", "identifier_type", "email_address"),
					resource.TestCheckResourceAttr("clerk_allowlist_identifier.test", "notify", "false"),
				),
			},
			{
				ResourceName:      "clerk_allowlist_identifier.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBlocklistIdentifier_basic(t *testing.T) {
	email := fmt.Sprintf("%s@example.com", acctest.RandomName("tf-acc"))
	config := fmt.Sprintf(`
resource "clerk_blocklist_identifier" "test" {
  identifier = %q
}
`, email)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkIdentifierDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_blocklist_identifier.test", "id"),
					resource.TestCheckResourceAttr("clerk_blocklist_identifier.test", "identifier", email),
					resource.TestCheckResourceAttr("clerk_blocklist_identifier.test", "identifier_type", "email_address"),
				),
			},
			{
				ResourceName:      "clerk_blocklist_identifier.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// checkIdentifierDestroy lists both identifier collections and fails when a
// tracked id is still present. The APIs have no get-by-id.
func checkIdentifierDestroy(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "clerk_allowlist_identifier":
			list, err := c.Allowlist.List(ctx, &sdkallowlist.ListParams{})
			if err != nil {
				return err
			}
			for _, entry := range list.AllowlistIdentifiers {
				if entry.ID == rs.Primary.ID {
					return fmt.Errorf("allowlist identifier %s still exists", rs.Primary.ID)
				}
			}
		case "clerk_blocklist_identifier":
			list, err := c.Blocklist.List(ctx, &sdkblocklist.ListParams{})
			if err != nil {
				return err
			}
			for _, entry := range list.BlocklistIdentifiers {
				if entry.ID == rs.Primary.ID {
					return fmt.Errorf("blocklist identifier %s still exists", rs.Primary.ID)
				}
			}
		}
	}
	return nil
}
