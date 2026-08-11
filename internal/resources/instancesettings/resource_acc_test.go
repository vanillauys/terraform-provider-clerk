package instancesettings_test

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccInstanceSettings_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	acctest.PreCheck(t)
	c, err := acctest.ClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// The resource mutates shared instance state: snapshot allowed_origins
	// and restore it after the test. An empty prior restores to [] which
	// clears the field — the same end state.
	prior, err := c.GetInstance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.UpdateAllowedOrigins(ctx, prior.AllowedOrigins); err != nil {
			t.Errorf("restoring allowed_origins: %v", err)
		}
	})

	config := func(origins string) string {
		return `
resource "clerk_instance_settings" "test" {
  test_mode       = true
  allowed_origins = ` + origins + `
}
`
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(`["https://acc.example.com"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_instance_settings.test", "id"),
					resource.TestCheckResourceAttr("clerk_instance_settings.test", "environment_type", "development"),
					resource.TestCheckResourceAttr("clerk_instance_settings.test", "allowed_origins.#", "1"),
				),
			},
			{
				Config: config(`["https://acc.example.com", "https://acc2.example.com"]`),
				Check:  resource.TestCheckResourceAttr("clerk_instance_settings.test", "allowed_origins.#", "2"),
			},
		},
	})
}
