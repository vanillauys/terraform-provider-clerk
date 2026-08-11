package apikey_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkuser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

// firstUserID picks an existing user of the dev instance as the key
// subject. The pick is read-only.
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
		t.Skip("the instance has no users; clerk_api_key needs a subject")
	}
	return list.Users[0].ID
}

func TestAccAPIKey_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	acctest.PreCheck(t)
	subject := firstUserID(t, context.Background())
	name := acctest.RandomName("tf-acc-key")

	config := func(extra string) string {
		return fmt.Sprintf(`
resource "clerk_api_key" "test" {
  name    = %q
  subject = %q
  scopes  = ["read"]
%s
}
`, name, subject, extra)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("clerk_api_key.test", "id"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "name", name),
					resource.TestCheckResourceAttr("clerk_api_key.test", "subject", subject),
					resource.TestCheckResourceAttr("clerk_api_key.test", "scopes.#", "1"),
					resource.TestCheckResourceAttrSet("clerk_api_key.test", "secret"),
					resource.TestCheckResourceAttr("clerk_api_key.test", "type", "api_key"),
				),
			},
			{
				Config: config(`  description = "acceptance test key"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clerk_api_key.test", "description", "acceptance test key"),
				),
			},
			{
				ResourceName:      "clerk_api_key.test",
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
		if rs.Type != "clerk_api_key" {
			continue
		}
		_, err := c.APIKeys.Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("API key %s still exists", rs.Primary.ID)
		}
		if !clerkapi.IsNotFound(err) {
			return err
		}
	}
	return nil
}
