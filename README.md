# terraform-provider-clerk

A Terraform provider for [Clerk](https://clerk.com). It manages the
configuration of a Clerk instance through the
[Clerk Backend API](https://clerk.com/docs/reference/backend-api).

**Status: pre-1.0.** All waves through SSO are on the
[Terraform Registry](https://registry.terraform.io/providers/vanillauys/clerk).
Breaking changes can land in minor releases until v1.0.0; the CHANGELOG
flags them.

## What it manages

| Area | Resources |
|------|-----------|
| Authentication config | `clerk_jwt_template`, `clerk_redirect_url`, `clerk_allowlist_identifier`, `clerk_blocklist_identifier` |
| Instance singletons | `clerk_instance_settings`, `clerk_instance_restrictions`, `clerk_instance_organization_settings` |
| Machine-to-machine | `clerk_api_key`, `clerk_machine` |
| Domains and webhooks | `clerk_domain`, `clerk_webhook` |
| Organizations | `clerk_organization`, `clerk_organization_permission`, `clerk_organization_role`, `clerk_organization_domain`, `clerk_organization_membership` |
| SSO | `clerk_oauth_application`, `clerk_saml_connection` |
| Data sources | `clerk_domains`, `clerk_jwks`, `clerk_organization` |

One provider block targets one Clerk instance. Use a provider alias for a
second instance (for example dev and prod). The
[guides](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/getting-started)
cover setup, aliases, adoption with import, and secrets.

## Example

```hcl
terraform {
  required_providers {
    clerk = {
      source = "vanillauys/clerk"
    }
  }
}

# Reads CLERK_SECRET_KEY when secret_key is not set.
provider "clerk" {}

resource "clerk_jwt_template" "api" {
  name     = "api"
  lifetime = 3600

  claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })
}

data "clerk_domains" "all" {}
```

## Provider configuration

| Attribute | Environment variable | Default |
|-----------|----------------------|---------|
| `secret_key` | `CLERK_SECRET_KEY` | required |
| `api_url` | `CLERK_API_URL` | `https://api.clerk.com/v1` |

The secret key is the instance key from the Clerk dashboard (`sk_test_...`
for a development instance, `sk_live_...` for production).

## Local development

Build the provider:

```sh
make build
```

Point Terraform at the local build with a dev override in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "vanillauys/clerk" = "/path/to/your/go/bin"
  }
  direct {}
}
```

Then run `go install .` and use the provider without `terraform init`.

## Tests

Run the unit tests:

```sh
make test
```

Run the acceptance tests. Warning: they mutate a real Clerk instance. Point
`CLERK_SECRET_KEY` at a development instance, never at production:

```sh
CLERK_SECRET_KEY=sk_test_... make testacc
```

## Docs

`make docs` renders the registry docs into `docs/` with tfplugindocs. CI
fails when the generated docs drift from the schema.

## Roadmap to v1.0.0

The provider grows in waves. Each wave ships as one tagged version with
acceptance tests, import support, and generated docs.

| Version | Wave | Status |
|---------|------|--------|
| v0.1.0 | Publish gate | Done |
| v0.2.0 | Instance plane | Done |
| v0.3.0 | API keys and machines | Done |
| v0.4.0 | Domains, webhooks, JWKS | Done |
| v0.5.0 | Organizations | Done |
| v0.6.0 | SSO | Done |
| v0.7.0 | Docs: the four guides | Done |
| v1.0.0 | Stable | After the dogfood gate: real stacks run on the registry build with an empty plan, and the nightly acceptance suite is green for 14 days. |

The runtime plane stays out of scope: users, sessions, tokens,
invitations, and the deprecated email/SMS template API. User lifecycle
fits Clerk better than Terraform.

## License

[MIT](LICENSE)
