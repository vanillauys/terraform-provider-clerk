package organization_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkinstancesettings "github.com/clerk/clerk-sdk-go/v2/instancesettings"
	sdkuser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func firstUserID(t *testing.T, ctx context.Context) string {
	t.Helper()
	cfg := &clerk.ClientConfig{}
	cfg.Key = clerk.String(os.Getenv("CLERK_SECRET_KEY"))
	if u := os.Getenv("CLERK_API_URL"); u != "" {
		cfg.URL = clerk.String(u)
	}
	list, err := sdkuser.NewClient(cfg).List(ctx, &sdkuser.ListParams{})
	if err != nil {
		t.Fatalf("listing users: %v", err)
	}
	if len(list.Users) == 0 {
		t.Skip("the instance has no users; the membership resource needs one")
	}
	return list.Users[0].ID
}

// TestAccOrganizationSuite drives the whole organizations ladder in one
// config: organization, custom permission, custom role, organization
// domain, and membership. The test enables the organizations feature
// (domains included) first and restores the prior settings in cleanup.
func TestAccOrganizationSuite(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	acctest.PreCheck(t)
	c, err := acctest.ClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	prior, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{
		Enabled:        clerk.Bool(true),
		DomainsEnabled: clerk.Bool(true),
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		modes := prior.DomainsEnrollmentModes
		_, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{
			Enabled:                clerk.Bool(prior.Enabled),
			DomainsEnabled:         clerk.Bool(prior.DomainsEnabled),
			DomainsEnrollmentModes: &modes,
		})
		if err != nil {
			t.Errorf("restoring instance organization settings: %v", err)
		}
	})

	user := firstUserID(t, ctx)
	name := acctest.RandomName("tf-acc-org")
	// Role and permission keys allow no dashes; derive a clean suffix.
	suffix := strings.ToLower(strings.ReplaceAll(acctest.RandomName("k"), "-", ""))
	domainName := fmt.Sprintf("%s.example.com", strings.ToLower(strings.ReplaceAll(acctest.RandomName("tf-acc"), "-", "")))

	config := func(orgName string) string {
		return fmt.Sprintf(`
resource "clerk_organization" "test" {
  name                    = %q
  max_allowed_memberships = 5

  public_metadata = jsonencode({
    env = "acc"
  })
}

resource "clerk_organization_permission" "test" {
  name        = "Acc Read %s"
  key         = "org:%s:read"
  description = "acceptance"
}

resource "clerk_organization_role" "test" {
  name        = "Acc Role %s"
  key         = "org:%s"
  permissions = [clerk_organization_permission.test.id]
}

resource "clerk_organization_domain" "test" {
  organization_id = clerk_organization.test.id
  name            = %q
  enrollment_mode = "manual_invitation"
}

resource "clerk_organization_membership" "test" {
  organization_id = clerk_organization.test.id
  user_id         = %q
  role            = clerk_organization_role.test.key
}
`, orgName, suffix, suffix, suffix, suffix, domainName, user)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_organization.test", "id"),
					resource.TestCheckResourceAttr("clerk_organization.test", "name", name),
					resource.TestCheckResourceAttrSet("clerk_organization.test", "slug"),
					resource.TestCheckResourceAttr("clerk_organization_permission.test", "type", "user"),
					resource.TestCheckResourceAttr("clerk_organization_role.test", "permissions.#", "1"),
					resource.TestCheckResourceAttr("clerk_organization_domain.test", "name", domainName),
					resource.TestCheckResourceAttrSet("clerk_organization_domain.test", "verification_status"),
					resource.TestCheckResourceAttr("clerk_organization_membership.test", "user_id", user),
					resource.TestCheckResourceAttrSet("clerk_organization_membership.test", "role_name"),
				),
			},
			{
				Config: config(name + "-renamed"),
				Check:  resource.TestCheckResourceAttr("clerk_organization.test", "name", name+"-renamed"),
			},
			{
				ResourceName:      "clerk_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "clerk_organization_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "clerk_organization_membership.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["clerk_organization_membership.test"]
					return rs.Primary.Attributes["organization_id"] + "/" + rs.Primary.Attributes["user_id"], nil
				},
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
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "clerk_organization":
			if _, err := c.Organizations.Get(ctx, rs.Primary.ID); err == nil {
				return fmt.Errorf("organization %s still exists", rs.Primary.ID)
			} else if !clerkapi.IsNotFound(err) {
				return err
			}
		case "clerk_organization_permission":
			if _, err := c.OrgPermissions.Get(ctx, rs.Primary.ID); err == nil {
				return fmt.Errorf("organization permission %s still exists", rs.Primary.ID)
			} else if !clerkapi.IsNotFound(err) {
				return err
			}
		case "clerk_organization_role":
			if _, err := c.OrgRoles.Get(ctx, rs.Primary.ID); err == nil {
				return fmt.Errorf("organization role %s still exists", rs.Primary.ID)
			} else if !clerkapi.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}
