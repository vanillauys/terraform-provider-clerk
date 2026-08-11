// Package acctest holds shared helpers for acceptance tests. The tests run
// against a real Clerk instance and mutate its configuration — point
// CLERK_SECRET_KEY at a development instance, never at production:
//
//	CLERK_SECRET_KEY=sk_test_... make testacc
package acctest

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/provider"
)

func PreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CLERK_SECRET_KEY") == "" {
		t.Fatal("CLERK_SECRET_KEY must be set for acceptance tests; use a development instance key")
	}
}

func ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"clerk": providerserver.NewProtocol6WithError(provider.New("acctest")()),
	}
}

// ClientFromEnv builds a direct API client for existence/destroy checks.
func ClientFromEnv() (*clerkapi.Client, error) {
	key := os.Getenv("CLERK_SECRET_KEY")
	if key == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY must be set")
	}
	return clerkapi.New(key, os.Getenv("CLERK_API_URL"), "acctest"), nil
}

// RandomName returns prefix-XXXXXXXX for collision-free test resources.
func RandomName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, sdkacctest.RandStringFromCharSet(8, sdkacctest.CharSetAlphaNum))
}
