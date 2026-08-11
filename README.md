# terraform-provider-clerk

A Terraform provider for [Clerk](https://clerk.com). It manages the
configuration of a Clerk instance through the
[Clerk Backend API](https://clerk.com/docs/reference/backend-api).

**Status: MVP.** The provider is not yet on the Terraform Registry. Build it
locally and use a dev override (see below). Breaking changes can land in
minor releases until v1.0.0.

## What it manages

| Type | Name | Operations |
|------|------|------------|
| Resource | `clerk_jwt_template` | create, read, update, delete, import |
| Resource | `clerk_redirect_url` | create, read, delete (replace on change), import |
| Resource | `clerk_allowlist_identifier` | create, read, delete (replace on change), import |
| Resource | `clerk_blocklist_identifier` | create, read, delete (replace on change), import |
| Data source | `clerk_domains` | read, with the CNAME records for DNS |

One provider block targets one Clerk instance. Use a provider alias for a
second instance (for example dev and prod).

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

## Roadmap

The Clerk Backend API exposes many more resource groups. Candidates, in
rough order of value:

| API group | Candidate | Note |
|-----------|-----------|------|
| Instance settings | `clerk_instance_settings` | The API has update calls but no read call; the resource cannot detect drift. |
| Domains | `clerk_domain` | Satellite domain management. |
| Webhooks | `clerk_webhook` | Svix-backed. |
| Organizations | `clerk_organization` + memberships, roles, permissions | |
| OAuth applications | `clerk_oauth_application` | |
| SAML connections | `clerk_saml_connection` | |
| API keys, machines, M2M | `clerk_api_key`, `clerk_machine` | Newer API surface. |
| Users, invitations | data sources first | User lifecycle fits Clerk better than Terraform. |

## License

[MIT](LICENSE)
