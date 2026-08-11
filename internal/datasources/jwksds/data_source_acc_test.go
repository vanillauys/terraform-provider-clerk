package jwksds_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccJWKSDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `data "clerk_jwks" "this" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.clerk_jwks.this", "keys.0.kid"),
					resource.TestCheckResourceAttrSet("data.clerk_jwks.this", "keys.0.algorithm"),
				),
			},
		},
	})
}
