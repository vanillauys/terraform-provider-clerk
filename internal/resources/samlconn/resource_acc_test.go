package samlconn_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vanillauys/terraform-provider-clerk/internal/acctest"
	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
)

func TestAccSAMLConnection_basic(t *testing.T) {
	name := acctest.RandomName("tf-acc-saml")
	// allow_subdomains needs an eTLD+1 domain, so the test domain has no
	// subdomain label.
	domain := fmt.Sprintf("%s.com", strings.ToLower(strings.ReplaceAll(name, "-", "")))
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	// active = true needs an IdP certificate.
	cert := selfSignedCert(t)
	acctest.PreCheck(t)
	acctest.EnableOrganizations(t)
	// linked sets organization_id and a non-default value for each flag.
	// Without it, the config drops them, so the step proves that a removed
	// organization_id reverts to null and each flag reverts to its default.
	config := func(name string, linked bool) string {
		org := ""
		if linked {
			org = `organization_id      = clerk_organization.test.id
  active               = true
  sync_user_attributes = false
  allow_subdomains     = true
  allow_idp_initiated  = true
  force_authn          = true`
		}
		return fmt.Sprintf(`
resource "clerk_organization" "test" {
  name = %[1]q
}

resource "clerk_saml_connection" "test" {
  name            = %[1]q
  domain          = %[2]q
  provider_key    = "saml_custom"
  idp_entity_id   = "https://idp.example.com/entity"
  idp_sso_url     = "https://idp.example.com/sso"
  idp_certificate = %[4]q
  %[3]s
}
`, name, domain, org, cert)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories(),
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("clerk_saml_connection.test", "organization_id", "clerk_organization.test", "id"),
					resource.TestCheckResourceAttrSet("clerk_saml_connection.test", "id"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "name", name),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "domain", domain),
					resource.TestCheckResourceAttrSet("clerk_saml_connection.test", "acs_url"),
					resource.TestCheckResourceAttrSet("clerk_saml_connection.test", "sp_entity_id"),
				),
			},
			{
				Config: config(name+"-renamed", true),
				Check:  resource.TestCheckResourceAttr("clerk_saml_connection.test", "name", name+"-renamed"),
			},
			{
				ResourceName:      "clerk_saml_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The import reads the bare base64 form; the state holds the PEM form.
				ImportStateVerifyIgnore: []string{"idp_certificate"},
			},
			{
				Config: config(name+"-renamed", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("clerk_saml_connection.test", "organization_id"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "active", "false"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "sync_user_attributes", "true"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "allow_subdomains", "false"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "allow_idp_initiated", "false"),
					resource.TestCheckResourceAttr("clerk_saml_connection.test", "force_authn", "false"),
					checkOrganizationCleared,
				),
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

// checkOrganizationCleared reads the connection from the API: a removed
// organization_id must be gone on the server, and each removed flag must
// hold its default there, not only in the state.
func checkOrganizationCleared(s *terraform.State) error {
	c, err := acctest.ClientFromEnv()
	if err != nil {
		return err
	}
	conn, err := c.SAMLConnections.Get(context.Background(), s.RootModule().Resources["clerk_saml_connection.test"].Primary.ID)
	if err != nil {
		return err
	}
	if conn.OrganizationID != nil && *conn.OrganizationID != "" {
		return fmt.Errorf("organization_id is %q on the server, want empty", *conn.OrganizationID)
	}
	if conn.Active || !conn.SyncUserAttributes || conn.AllowSubdomains || conn.AllowIdpInitiated || conn.ForceAuthn {
		return fmt.Errorf("connection holds active=%t sync=%t subdomains=%t idp_initiated=%t force_authn=%t, want the defaults",
			conn.Active, conn.SyncUserAttributes, conn.AllowSubdomains, conn.AllowIdpInitiated, conn.ForceAuthn)
	}
	return nil
}

// selfSignedCert returns a PEM X.509 certificate for the IdP side of the test.
func selfSignedCert(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "tf-acc-idp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
