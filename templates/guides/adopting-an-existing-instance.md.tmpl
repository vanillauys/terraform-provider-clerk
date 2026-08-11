---
page_title: "Adopting an existing instance"
subcategory: ""
description: |-
  Bring a configured Clerk instance under Terraform with import, without recreation.
---

# Adopting an existing instance

Most instances already carry configuration from the dashboard. This guide
brings that configuration under Terraform by **import**, never by
recreation.

## The singletons adopt themselves

The three instance singletons need no import. Create adopts the instance:

```terraform
resource "clerk_instance_restrictions" "this" {
  block_disposable_email_domains = true
}
```

The first apply patches only the fields you set. You can also import them
with any id (`terraform import clerk_instance_restrictions.this instance`);
the first refresh replaces the id with the real instance id.

`clerk_instance_restrictions` and `clerk_instance_organization_settings`
read their full server state, so a plan after import shows real drift.
`clerk_instance_settings` cannot: most of its fields are write-only (see
the resource docs), and the first apply simply asserts your configuration.

## Import ids per resource

| Resource | Import id |
|----------|-----------|
| clerk_jwt_template | `jtmp_...` |
| clerk_redirect_url | `ru_...` |
| clerk_allowlist_identifier / clerk_blocklist_identifier | `alid_...` / `blid_...` |
| clerk_api_key | `ak_...` (the id, not the secret) |
| clerk_machine | `mch_...` |
| clerk_domain | `dmn_...` |
| clerk_organization | `org_...` or the slug |
| clerk_organization_permission / _role | `perm_...` / `role_...` |
| clerk_organization_domain | `org_.../orgdmn_...` |
| clerk_organization_membership | `org_.../user_...` |
| clerk_oauth_application | the application id |
| clerk_saml_connection | `samlc_...` |

List the ids with the [Clerk CLI](https://clerk.com/docs/cli)
(`clerk api`) or with `curl` against the Backend API.

## What an import cannot recover

Write-only values stay null after an import:

- `clerk_jwt_template.signing_key`
- `clerk_oauth_application.client_secret` (Clerk returns it only on create)
- `clerk_oauth_application.scopes` and `clerk_allowlist_identifier.notify`
- The write-only fields of `clerk_instance_settings`

After the import, the first plan shows the configured values of those
fields as in-place changes. That apply converges the state; it does not
recreate the resource. API-key secrets and machine secret keys are the
exception: Read retrieves them, so an import recovers them completely.

## clerk_webhook has no import

The Svix integration has no read endpoint. If Svix is already on, add the
resource and apply once: the enable call is idempotent server-side.
