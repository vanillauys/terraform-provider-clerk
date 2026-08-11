package jwttemplate_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccJWTTemplate_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-jwt")
	config := func(lifetime int) string {
		return fmt.Sprintf(`
resource "clerk_jwt_template" "test" {
  name     = %q
  lifetime = %d

  claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })
}
`, name, lifetime)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_jwt_template.test", "id"),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "name", name),
					resource.TestCheckResourceAttr("clerk_jwt_template.test", "lifetime", "3600"),
					resource.TestCheckResourceAttrSet("clerk_jwt_template.test", "signing_algorithm"),
				),
			},
			{
				Config: config(7200),
				Check:  resource.TestCheckResourceAttr("clerk_jwt_template.test", "lifetime", "7200"),
			},
			{
				ResourceName:            "clerk_jwt_template.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"signing_key"},
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
