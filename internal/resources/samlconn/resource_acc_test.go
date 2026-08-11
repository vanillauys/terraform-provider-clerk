package samlconn_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccSAMLConnection_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-saml")
	domain := fmt.Sprintf("%s.example.com", strings.ToLower(strings.ReplaceAll(name, "-", "")))
	config := func(name string) string {
		return fmt.Sprintf(`
resource "clerk_saml_connection" "test" {
  name          = %q
  domain        = %q
  provider_key  = "saml_custom"
  idp_entity_id = "https://idp.example.com/entity"
  idp_sso_url   = "https://idp.example.com/sso"
}
`, name, domain)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_saml_connection.test", "id"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "name", name),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "domain", domain),
					resource.TestCheckResourceAttrSet("clerk_saml_connection.test", "acs_url"),
					resource.TestCheckResourceAttrSet("clerk_saml_connection.test", "sp_entity_id"),
				),
			},
			{
				Config: config(name + "-renamed"),
				Check:  resource.TestCheckResourceAttr("clerk_saml_connection.test", "name", name+"-renamed"),
			},
			{
				ResourceName:      "clerk_saml_connection.test",
				ImportState:       true,
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
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "clerk_saml_connection" {
			continue
		}
		_, err := c.SAMLConnections.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("SAML connection %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}
