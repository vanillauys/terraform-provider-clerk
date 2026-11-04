package organization_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	sdkorgdomain "github.com/clerk/clerk-sdk-go/v2/organizationdomain"
	sdkorgmembership "github.com/clerk/clerk-sdk-go/v2/organizationmembership"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

// TestAccOrganizationSuite drives the whole organizations ladder in one
// config: organization, custom permission, custom role, organization
// domain, and membership. The test enables the organizations feature
// (domains included) first and restores the prior settings in cleanup.
func TestAccOrganizationSuite(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	acctest.PreCheck(t)
	acctest.EnableOrganizations(t)

	user := acctest.TestUser(t)
	// creator holds created_by: Clerk adds that user as the first admin.
	creator := acctest.TestUser(t)
	name := acctest.RandomName("tf-acc-org")
	// Role and permission keys allow no dashes; derive a clean suffix.
	suffix := strings.ToLower(strings.ReplaceAll(acctest.RandomName("k"), "-", ""))
	domainName := fmt.Sprintf("%s.example.com", strings.ToLower(strings.ReplaceAll(acctest.RandomName("tf-acc"), "-", "")))

	// full adds the descriptions, the custom member role, and a non-default
	// value for each optional attribute. Without it, the config drops them and
	// moves the member to org:member, so the step proves that a removed
	// description reverts to null and each other attribute to its default.
	config := func(orgName string, full bool) string {
		description, role, enrollment, orgExtra, permissions := "", `"org:member"`, "", "", ""
		if full {
			description = `description = "acceptance"`
			role = "clerk_organization_role.test.key"
			enrollment = `enrollment_mode = "automatic_invitation"
  verified        = true`
			orgExtra = `admin_delete_enabled = false
  public_metadata      = jsonencode({ env = "acc" })
  private_metadata     = jsonencode({ team = "acc" })`
			permissions = "permissions = [clerk_organization_permission.test.id]"
		}
		return fmt.Sprintf(`
resource "clerk_organization" "test" {
  name                    = %[1]q
  max_allowed_memberships = 5
  created_by              = %[10]q
  %[8]s
}

resource "clerk_organization_permission" "test" {
  name = "Acc Read %[2]s"
  key  = "org:%[2]s:read"
  %[3]s
}

resource "clerk_organization_role" "test" {
  name        = "Acc Role %[2]s"
  key         = "org:%[2]s"
  %[9]s
  %[3]s
}

resource "clerk_organization_domain" "test" {
  organization_id = clerk_organization.test.id
  name            = %[4]q
  %[7]s
}

resource "clerk_organization_membership" "test" {
  organization_id = clerk_organization.test.id
  user_id         = %[5]q
  role            = %[6]s
}

data "clerk_organization" "by_slug" {
  lookup = clerk_organization.test.slug
}

data "clerk_organizations" "match" {
  query = clerk_organization.test.name
}
`, orgName, suffix, description, domainName, user, role, enrollment, orgExtra, permissions, creator)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_organization_permission.test", "description", "acceptance"),
					resource.TestCheckResourceAttr("clerk_organization_role.test", "description", "acceptance"),
					resource.TestCheckResourceAttrPair("data.clerk_organization.by_slug", "id", "clerk_organization.test", "id"),
					resource.TestCheckResourceAttr("data.clerk_organization.by_slug", "max_allowed_memberships", "5"),
					resource.TestCheckResourceAttr("data.clerk_organizations.match", "organizations.#", "1"),
					resource.TestCheckResourceAttrPair("data.clerk_organizations.match", "organizations.0.id", "clerk_organization.test", "id"),
					resource.TestCheckResourceAttrSet("clerk_organization.test", "id"),
					resource.TestCheckResourceAttr("clerk_organization.test", "name", name),
					resource.TestCheckResourceAttrSet("clerk_organization.test", "slug"),
					resource.TestCheckResourceAttr("clerk_organization_permission.test", "type", "user"),
					resource.TestCheckResourceAttr("clerk_organization_role.test", "permissions.#", "1"),
					resource.TestCheckResourceAttr("clerk_organization_domain.test", "name", domainName),
					resource.TestCheckResourceAttr("clerk_organization_domain.test", "verification_status", "verified"),
					resource.TestCheckResourceAttr("clerk_organization.test", "created_by", creator),
					resource.TestCheckResourceAttr("clerk_organization_membership.test", "user_id", user),
					resource.TestCheckResourceAttrSet("clerk_organization_membership.test", "role_name"),
				),
			},
			{
				Config: config(name+"-renamed", true),
				Check:  resource.TestCheckResourceAttr("clerk_organization.test", "name", name+"-renamed"),
			},
			{
				ResourceName:      "clerk_organization_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: "clerk_organization_domain.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["clerk_organization_domain.test"]
					return rs.Primary.Attributes["organization_id"] + "/" + rs.Primary.ID, nil
				},
				ImportStateVerify: true,
				// verified is write-only: Clerk reports verification_status.
				ImportStateVerifyIgnore: []string{"verified"},
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
			{
				Config: config(name+"-renamed", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("clerk_organization_permission.test", "description"),
					resource.TestCheckNoResourceAttr("clerk_organization_role.test", "description"),
					resource.TestCheckResourceAttr("clerk_organization_membership.test", "role", "org:member"),
					resource.TestCheckResourceAttr("clerk_organization_domain.test", "enrollment_mode", "manual_invitation"),
					resource.TestCheckResourceAttr("clerk_organization_role.test", "permissions.#", "0"),
					resource.TestCheckResourceAttr("clerk_organization.test", "admin_delete_enabled", "true"),
					resource.TestCheckResourceAttr("clerk_organization.test", "public_metadata", "{}"),
					resource.TestCheckResourceAttr("clerk_organization.test", "private_metadata", "{}"),
					checkDescriptionsCleared,
					checkOrganizationDefaults,
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
		case "clerk_organization_domain":
			// The organization is gone too, so the list returns 404.
			list, err := c.OrgDomains.List(ctx, rs.Primary.Attributes["organization_id"], &sdkorgdomain.ListParams{})
			if clerkapi.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			for _, d := range list.OrganizationDomains {
				if d.ID == rs.Primary.ID {
					return fmt.Errorf("organization domain %s still exists", rs.Primary.ID)
				}
			}
		case "clerk_organization_membership":
			list, err := c.OrgMemberships.List(ctx, &sdkorgmembership.ListParams{
				OrganizationID: rs.Primary.Attributes["organization_id"],
				UserIDs:        []string{rs.Primary.Attributes["user_id"]},
			})
			if clerkapi.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if len(list.OrganizationMemberships) > 0 {
				return fmt.Errorf("organization membership %s still exists", rs.Primary.ID)
			}
		}
	}
	return nil
}

// checkDescriptionsCleared reads the permission and the role from the API:
// a removed description must be gone on the server, not only in the state.
func checkDescriptionsCleared(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	ctx := context.Background()
	perm, err := c.OrgPermissions.Get(ctx, s.RootModule().Resources["clerk_organization_permission.test"].Primary.ID)
	if err != nil {
		return err
	}
	if perm.Description != nil && *perm.Description != "" {
		return fmt.Errorf("permission description is %q, want empty", *perm.Description)
	}
	role, err := c.OrgRoles.Get(ctx, s.RootModule().Resources["clerk_organization_role.test"].Primary.ID)
	if err != nil {
		return err
	}
	if role.Description != nil && *role.Description != "" {
		return fmt.Errorf("role description is %q, want empty", *role.Description)
	}
	return nil
}

// checkOrganizationDefaults reads the organization and the role from the
// API: the removed attributes must hold the defaults on the server, not only
// in the state.
func checkOrganizationDefaults(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	ctx := context.Background()
	org, err := c.Organizations.Get(ctx, s.RootModule().Resources["clerk_organization.test"].Primary.ID)
	if err != nil {
		return err
	}
	if !org.AdminDeleteEnabled {
		return fmt.Errorf("admin_delete_enabled is false on the server, want true")
	}
	for field, raw := range map[string]string{"public_metadata": string(org.PublicMetadata), "private_metadata": string(org.PrivateMetadata)} {
		if raw != "{}" && raw != "" && raw != "null" {
			return fmt.Errorf("%s is %s on the server, want {}", field, raw)
		}
	}
	role, err := c.OrgRoles.Get(ctx, s.RootModule().Resources["clerk_organization_role.test"].Primary.ID)
	if err != nil {
		return err
	}
	if len(role.Permissions) != 0 {
		return fmt.Errorf("role has %d permissions on the server, want 0", len(role.Permissions))
	}
	return nil
}
