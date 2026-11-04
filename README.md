# Terraform provider for Clerk

[![Terraform Registry](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fregistry.terraform.io%2Fv1%2Fproviders%2Fvanillauys%2Fclerk&query=%24.version&prefix=v&label=registry&color=7B42BC&logo=terraform&logoColor=white)](https://registry.terraform.io/providers/vanillauys/clerk/latest)
[![Registry downloads](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fregistry.terraform.io%2Fv2%2Fproviders%2Fvanillauys%2Fclerk&query=%24.data.attributes.downloads&label=downloads&color=7B42BC)](https://registry.terraform.io/providers/vanillauys/clerk/latest)
[![Clerk API 2026-05-12](https://img.shields.io/badge/Clerk%20API-2026--05--12-6C47FF)](https://clerk.com/docs/reference/backend-api)
[![test](https://github.com/vanillauys/terraform-provider-clerk/actions/workflows/test.yml/badge.svg)](https://github.com/vanillauys/terraform-provider-clerk/actions/workflows/test.yml)
[![acceptance](https://github.com/vanillauys/terraform-provider-clerk/actions/workflows/acceptance.yml/badge.svg)](https://github.com/vanillauys/terraform-provider-clerk/actions/workflows/acceptance.yml)
[![nightly acceptance](https://github.com/vanillauys/terraform-provider-clerk/actions/workflows/nightly.yml/badge.svg)](https://github.com/vanillauys/terraform-provider-clerk/actions/workflows/nightly.yml)
[![Quality gate](https://sonarcloud.io/api/project_badges/measure?project=vanillauys_terraform-provider-clerk&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=vanillauys_terraform-provider-clerk)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/vanillauys/terraform-provider-clerk/badge)](https://scorecard.dev/viewer/?uri=github.com/vanillauys/terraform-provider-clerk)
[![License: MIT](https://img.shields.io/github/license/vanillauys/terraform-provider-clerk)](LICENSE)

Manage the configuration of a [Clerk](https://clerk.com) instance as code,
with Terraform or OpenTofu. Declare your JWT templates, sign-up rules,
domains, organizations, roles, and SSO connections. `terraform apply`
creates them, and the next plan shows each change that someone made in the
dashboard.

- **18 resources and 4 data sources** for the
  [Clerk Backend API](https://clerk.com/docs/reference/backend-api). Each one
  has an acceptance test that runs against a real Clerk development instance
  on every pull request and every night.
- **Explicit defaults.** When you remove an optional attribute, it reverts
  to the Clerk default. An acceptance step reads the record from the API to
  prove each revert.
- **Secrets stay out of the state.** The JWT template signing key has a
  write-only form (`signing_key_wo`) for Terraform 1.11 or later.
- **Dev and prod in one plan.** One provider block targets one instance, and
  an alias targets the next.
- **Adoption.** Every resource except `clerk_webhook` imports, so you can
  bring a configured instance under Terraform without a rebuild.

[Documentation](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs) ·
[Get started](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/getting-started) ·
[Changelog](CHANGELOG.md) ·
[Contributing](CONTRIBUTING.md) ·
[Security](SECURITY.md)

## Quick start

1. In the Clerk dashboard, open **Configure > API keys** and copy the
   secret key of a development instance (`sk_test_...`).

2. Export the key:

   ```sh
   export CLERK_SECRET_KEY="sk_test_..."
   ```

3. Save this configuration as `main.tf`. It creates a JWT template with a
   custom claim and an allowed redirect URL:

   ```hcl
   terraform {
     required_providers {
       clerk = {
         source  = "vanillauys/clerk"
         version = "~> 0.7"
       }
     }
   }

   # The provider reads CLERK_SECRET_KEY.
   provider "clerk" {}

   resource "clerk_jwt_template" "api" {
     name     = "api"
     lifetime = 3600

     claims = jsonencode({
       role = "{{user.public_metadata.role}}"
     })
   }

   resource "clerk_redirect_url" "app" {
     url = "https://app.example.com/callback"
   }
   ```

4. Apply it:

   ```sh
   terraform init
   terraform apply
   ```

The [Get started](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/getting-started)
guide continues with sign-up restrictions and a second instance.

## Provider configuration

| Argument | Environment variable | Required | Description |
|----------|----------------------|----------|-------------|
| `secret_key` | `CLERK_SECRET_KEY` | yes | The secret key of the instance: `sk_test_...` for development, `sk_live_...` for production. Sensitive. |
| `api_url` | `CLERK_API_URL` | no | The Backend API URL. Defaults to `https://api.clerk.com/v1`. |

Set the key through the environment variable, not in the configuration, so
that it stays out of your repository.

## Compatibility

- **Clerk Backend API:** the provider sends the API version that
  clerk-sdk-go pins, in the `Clerk-API-Version` header. clerk-sdk-go v2.7.0
  pins `2026-05-12`. An SDK upgrade that moves the API version gets its own
  minor release and a CHANGELOG note.
- **Terraform:** 1.5 or later. The write-only attribute `signing_key_wo`
  needs 1.11 or later. The nightly run tests Terraform 1.5.7 and the latest
  release.
- **OpenTofu:** the nightly run tests OpenTofu 1.12.9 with the Terraform
  Registry address `vanillauys/clerk`. The provider is not on the OpenTofu
  registry yet.

## What you can manage

| Area | Resources |
|------|-----------|
| Authentication | [`clerk_jwt_template`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/jwt_template), [`clerk_redirect_url`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/redirect_url), [`clerk_allowlist_identifier`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/allowlist_identifier), [`clerk_blocklist_identifier`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/blocklist_identifier) |
| Instance | [`clerk_instance_settings`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/instance_settings), [`clerk_instance_restrictions`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/instance_restrictions), [`clerk_instance_organization_settings`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/instance_organization_settings) |
| Machine-to-machine | [`clerk_api_key`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/api_key), [`clerk_machine`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/machine) |
| Domains and webhooks | [`clerk_domain`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/domain), [`clerk_webhook`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/webhook) |
| Organizations | [`clerk_organization`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/organization), [`clerk_organization_permission`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/organization_permission), [`clerk_organization_role`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/organization_role), [`clerk_organization_domain`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/organization_domain), [`clerk_organization_membership`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/organization_membership) |
| SSO | [`clerk_oauth_application`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/oauth_application), [`clerk_saml_connection`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/resources/saml_connection) |
| Data sources | [`clerk_domains`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/data-sources/domains), [`clerk_jwks`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/data-sources/jwks), [`clerk_organization`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/data-sources/organization), [`clerk_organizations`](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/data-sources/organizations) |

The runtime plane stays out of scope: users, sessions, tokens, invitations,
and the deprecated email and SMS template API. Clerk manages the user
lifecycle better than Terraform.

## Before you start

1. **The singletons adopt, they never delete.** The instance resources
   (settings, restrictions, organization settings, webhook) take over the
   instance on create. A destroy removes them from the state only.
2. **Some fields are write-only.** Clerk does not return them, so the
   provider cannot see a dashboard change on them. Each resource page names
   its write-only fields.
3. **The state holds secrets:** API key secrets, machine secret keys, and
   OAuth client secrets. Protect the state like a secret. See the
   [Secrets guide](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/secrets).

## Guides

- **[Get started](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/getting-started)**: configure the provider and apply a first JWT template, redirect URL, and restriction.
- **[Dev and prod instances](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/dev-and-prod-instances)**: one plan, two instances, provider aliases, and the DNS records.
- **[Adopt an existing instance](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/adopting-an-existing-instance)**: bring a configured instance under Terraform with import.
- **[Secrets and sensitive values](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/secrets)**: what goes into the state and how to protect it.

## Known limitations

The provider cannot do these things. In most cases, Clerk has no API for
them.

- **A primary domain cannot change.** The Backend API creates satellite
  domains only. Change the primary domain in the dashboard.
- **`clerk_webhook` cannot detect drift or import.** Clerk has no read
  endpoint for the Svix integration.
- **Some attributes keep their value when you remove them**, because Clerk
  derives the value and no fixed default exists: the `clerk_organization`
  `slug`, `max_allowed_memberships`, and `created_by`, the
  `clerk_saml_connection` `idp_` fields, the `clerk_jwt_template`
  `signing_key`, and the `clerk_api_key` `seconds_until_expiration`. Each
  attribute description says what occurs.
- **An attribute that you remove from a singleton** stops under Terraform
  management, and Clerk keeps its value.
- **`clerk_api_key` needs the API Keys feature** of the instance. Turn it on
  in the dashboard first.

## Stability

**Status: pre-1.0.** v1.0.0 follows the dogfood gate: real stacks run on the
registry build with an empty plan, and the nightly acceptance suite is green
for 14 days. Until then, a minor release can have a breaking change, and the
CHANGELOG flags it.

From v1.0.0, the provider follows [semantic versioning](https://semver.org):

- A minor release adds resources, data sources, and attributes. A
  configuration and a state from the previous minor release load with an
  empty plan. `TestAccUpgradeFromPrevious` proves it on each pull request.
- A change that removes or renames an attribute, changes a default, or
  changes what an existing attribute does needs a major release. The schema
  snapshot test (`internal/provider/testdata/schema.json`) fails on such a
  change.

Pin the minor version, for example `~> 1.0`, to get fixes and additions
without a breaking change.

## Contributing

Bug reports, feature requests, and pull requests are welcome.
[CONTRIBUTING.md](CONTRIBUTING.md) walks you from the fork to the merge.
Know these three rules first:

- **Every commit must be signed** and show **Verified** on GitHub. The
  guide shows the SSH and the GPG setup.
- **Open an issue before you add a resource or an attribute**, so that the
  shape is agreed first.
- **Acceptance tests run against a Clerk development instance.** Never point
  them at a production instance.

Quick reference:

```sh
make test    # unit tests and the schema snapshot
make lint    # golangci-lint, must print "0 issues"
make docs    # regenerate docs/ from templates/ and examples/
CLERK_SECRET_KEY=sk_test_... make testacc
```

## Security

Report a vulnerability through
[GitHub private vulnerability reporting](https://github.com/vanillauys/terraform-provider-clerk/security/advisories/new),
not through a public issue. [SECURITY.md](SECURITY.md) states the scope and
the supported versions. Each release is signed with GPG key
`750EE4482941313E`, the key that the Terraform Registry verifies, and
carries a GitHub build provenance attestation.

## Support

This is a community project, not an official Clerk product. Ask a question
or report a problem in the
[issues](https://github.com/vanillauys/terraform-provider-clerk/issues).
Help comes on a best-effort basis.

## License

[MIT](LICENSE).
