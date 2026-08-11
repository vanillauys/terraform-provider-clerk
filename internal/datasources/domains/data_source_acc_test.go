package domains_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
)

func TestAccDomainsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `data "clerk_domains" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.clerk_domains.all", "domains.0.id"),
					resource.TestCheckResourceAttrSet("data.clerk_domains.all", "domains.0.name"),
					resource.TestCheckResourceAttrSet("data.clerk_domains.all", "domains.0.frontend_api_url"),
				),
			},
		},
	})
}
