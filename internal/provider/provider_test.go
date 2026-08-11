package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveConfig(t *testing.T) {
	env := map[string]string{
		"CLERK_SECRET_KEY": "sk_env",
		"CLERK_API_URL":    "https://env.example.com/v1",
	}
	getenv := func(k string) string { return env[k] }
	empty := func(string) string { return "" }

	t.Run("explicit config wins over env", func(t *testing.T) {
		m := ClerkProviderModel{
			SecretKey: types.StringValue("sk_explicit"),
			APIURL:    types.StringValue("https://explicit.example.com/v1"),
		}
		rc, missing := resolveConfig(m, getenv)
		if len(missing) != 0 {
			t.Fatalf("missing = %v, want none", missing)
		}
		if rc.secretKey != "sk_explicit" || rc.apiURL != "https://explicit.example.com/v1" {
			t.Fatalf("rc = %+v", rc)
		}
	})

	t.Run("env fills the gaps", func(t *testing.T) {
		rc, missing := resolveConfig(ClerkProviderModel{}, getenv)
		if len(missing) != 0 {
			t.Fatalf("missing = %v, want none", missing)
		}
		if rc.secretKey != "sk_env" || rc.apiURL != "https://env.example.com/v1" {
			t.Fatalf("rc = %+v", rc)
		}
	})

	t.Run("missing secret key is reported", func(t *testing.T) {
		rc, missing := resolveConfig(ClerkProviderModel{}, empty)
		if len(missing) != 1 {
			t.Fatalf("missing = %v, want one entry", missing)
		}
		if rc.apiURL != "" {
			t.Fatalf("apiURL = %q, want empty (SDK default applies)", rc.apiURL)
		}
	})
}
