package machine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccMachine_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-machine")
	config := func(body string) string {
		return fmt.Sprintf(`
resource "clerk_machine" "target" {
  name = "%s-target"
}

resource "clerk_machine" "test" {
  name = %q
  %s
}
`, name, name, body)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config("scoped_machines = [clerk_machine.target.id]\n  default_token_ttl = 7200"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_machine.test", "id"),
					resource.TestCheckResourceAttr("clerk_machine.test", "name", name),
					resource.TestCheckResourceAttr("clerk_machine.test", "scoped_machines.#", "1"),
					resource.TestCheckResourceAttrSet("clerk_machine.test", "secret_key"),
					resource.TestCheckResourceAttr("clerk_machine.test", "default_token_ttl", "7200"),
				),
			},
			{
				// Drop the scope: reconcile must delete it server-side.
				Config: config("scoped_machines = []\n  default_token_ttl = 7200"),
				Check:  resource.TestCheckResourceAttr("clerk_machine.test", "scoped_machines.#", "0"),
			},
			{
				ResourceName:      "clerk_machine.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config("scoped_machines = [clerk_machine.target.id]\n  default_token_ttl = 7200"),
				Check:  resource.TestCheckResourceAttr("clerk_machine.test", "scoped_machines.#", "1"),
			},
			{
				// Remove both optional attributes: each one reverts to its default.
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_machine.test", "default_token_ttl", "3600"),
					resource.TestCheckResourceAttr("clerk_machine.test", "scoped_machines.#", "0"),
					checkDefaults,
				),
			},
		},
	})
}

// checkDefaults reads the machine from the API: the removed attributes must
// hold the defaults on the server, not only in the state.
func checkDefaults(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	m, err := c.Machines.Get(context.Background(), s.RootModule().Resources["clerk_machine.test"].Primary.ID)
	if err != nil {
		return err
	}
	if m.DefaultTokenTTL != 3600 || len(m.ScopedMachines) != 0 {
		return fmt.Errorf("machine holds ttl=%d scoped=%d, want 3600 and 0", m.DefaultTokenTTL, len(m.ScopedMachines))
	}
	return nil
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
