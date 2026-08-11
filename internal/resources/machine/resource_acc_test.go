package machine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccMachine_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-machine")
	config := func(scoped string) string {
		return fmt.Sprintf(`
resource "clerk_machine" "target" {
  name = "%s-target"
}

resource "clerk_machine" "test" {
  name            = %q
  scoped_machines = %s
}
`, name, name, scoped)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(`[clerk_machine.target.id]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_machine.test", "id"),
					resource.TestCheckResourceAttr("clerk_machine.test", "name", name),
					resource.TestCheckResourceAttr("clerk_machine.test", "scoped_machines.#", "1"),
					resource.TestCheckResourceAttrSet("clerk_machine.test", "secret_key"),
					resource.TestCheckResourceAttrSet("clerk_machine.test", "default_token_ttl"),
				),
			},
			{
				// Drop the scope: reconcile must delete it server-side.
				Config: config(`[]`),
				Check:  resource.TestCheckResourceAttr("clerk_machine.test", "scoped_machines.#", "0"),
			},
			{
				ResourceName:      "clerk_machine.test",
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
		if rs.Type != "clerk_machine" {
			continue
		}
		_, err := c.Machines.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("machine %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}
