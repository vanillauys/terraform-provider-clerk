// Package acctest holds shared helpers for acceptance tests. The tests run
// against a real Clerk instance and mutate its configuration — point
// CLERK_SECRET_KEY at a development instance, never at production:
//
//	CLERK_SECRET_KEY=sk_test_... make testacc
package acctest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkinstancesettings "github.com/clerk/clerk-sdk-go/v2/instancesettings"
	sdkuser "github.com/clerk/clerk-sdk-go/v2/user"
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

// TestUser creates a throwaway user for the resources that need a subject or
// a member, and deletes it when the test ends. The user has a username, a
// phone number, and an email address, so it meets the strictest sign-up
// requirements of an instance. On a development instance, the `+clerk_test`
// subaddress and the 555-01xx numbers are test values: Clerk sends no message.
func TestUser(t *testing.T) string {
	t.Helper()
	cfg := &clerk.ClientConfig{}
	cfg.Key = clerk.String(os.Getenv("CLERK_SECRET_KEY"))
	if u := os.Getenv("CLERK_API_URL"); u != "" {
		cfg.URL = clerk.String(u)
	}
	users := sdkuser.NewClient(cfg)
	ctx := context.Background()
	var u *clerk.User
	var err error
	// Clerk reserves +1 XXX 555-0100 to 555-0199 as test phone numbers, so
	// only 100 numbers exist. A user that leaked from a cancelled run can
	// hold one; try again with a new number on a conflict.
	for attempt := 0; attempt < 5; attempt++ {
		username := strings.ToLower(RandomName("tf-acc"))
		phone := fmt.Sprintf("+1201555%04d", sdkacctest.RandIntRange(100, 200))
		u, err = users.Create(ctx, &sdkuser.CreateParams{
			Username:                clerk.String(username),
			PhoneNumbers:            &[]string{phone},
			EmailAddresses:          &[]string{username + "+clerk_test@example.com"},
			SkipPasswordRequirement: clerk.Bool(true),
			SkipLegalChecks:         clerk.Bool(true),
		})
		var apiErr *clerk.APIErrorResponse
		if err == nil || !errors.As(err, &apiErr) || apiErr.HTTPStatusCode != 422 {
			break
		}
	}
	if err != nil {
		t.Fatalf("creating the test user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := users.Delete(ctx, u.ID); err != nil && !clerkapi.IsNotFound(err) {
			t.Errorf("deleting the test user %s: %v", u.ID, err)
		}
	})
	return u.ID
}

// EnableOrganizations turns on the organizations feature (domains included)
// for the test, and restores the prior settings when the test ends.
func EnableOrganizations(t *testing.T) {
	t.Helper()
	c, err := ClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	prior, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{
		Enabled:                clerk.Bool(true),
		DomainsEnabled:         clerk.Bool(true),
		DomainsEnrollmentModes: &[]string{"manual_invitation", "automatic_invitation", "automatic_suggestion"},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		modes := prior.DomainsEnrollmentModes
		_, err := c.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{
			Enabled:                clerk.Bool(prior.Enabled),
			DomainsEnabled:         clerk.Bool(prior.DomainsEnabled),
			DomainsEnrollmentModes: &modes,
		})
		if err != nil {
			t.Errorf("restoring instance organization settings: %v", err)
		}
	})
}
