package webhook_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

// The dev instance has Svix off; the test enables it and the destroy at
// the end of the test disables it again — the prior state returns.
func TestAccWebhook_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `resource "clerk_webhook" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_webhook.test", "id"),
					resource.TestCheckResourceAttrSet("clerk_webhook.test", "svix_url"),
				),
			},
		},
	})
}
