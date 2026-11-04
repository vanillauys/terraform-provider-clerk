---
page_title: "Secrets and sensitive values"
subcategory: ""
description: |-
  Which secrets the provider touches, what lands in state, and how to treat both.
---

# Secrets and sensitive values

## The provider credential

The `secret_key` of the provider is the most powerful credential: it
controls the whole instance. Keep it out of committed files. Two safe
patterns:

```sh
# From the environment (for example through a secret manager):
CLERK_SECRET_KEY=$(op read "op://vault/item/secret_key") terraform apply
```

```terraform
# Or through a sensitive variable:
variable "clerk_secret_key" {
  type      = string
  sensitive = true
}
```

## Secrets that land in state

Terraform state stores every attribute value, sensitive ones included.
These attributes carry real secrets:

| Attribute | Behavior |
|-----------|----------|
| `clerk_api_key.secret` | Read keeps it fresh; an import recovers it. |
| `clerk_machine.secret_key` | Same. |
| `clerk_oauth_application.client_secret` | Create-once; an import cannot recover it. |
| `clerk_jwt_template.signing_key` | Clerk never returns it; the state keeps your input. Use `signing_key_wo` to keep it out. |
| `clerk_webhook.svix_url` | A short-lived dashboard login link. |

The provider marks them `sensitive`, so plans redact them. The state file
does not: treat the state itself as a secret. Store it encrypted, restrict
access to it, and never commit it.

## Keep the signing key out of the state

With Terraform 1.11 or later, set the write-only `signing_key_wo` instead of
`signing_key`. Terraform sends it to Clerk but keeps it out of the plan and
the state. The plan cannot show a new write-only value, so change
`signing_key_wo_version` to send a new key:

```terraform
resource "clerk_jwt_template" "api" {
  name                   = "api"
  custom_signing_key     = true
  signing_algorithm      = "RS256"
  signing_key_wo         = var.jwt_signing_key
  signing_key_wo_version = 1
}
```

## Consume secrets from other resources

Reference the attributes directly; Terraform keeps the value inside the
graph:

```terraform
resource "clerk_api_key" "backend" {
  name    = "backend"
  subject = var.service_user_id
}

# Example: hand the key to the platform that runs your API.
resource "dokploy_application" "api" {
  # ...
  env = "CLERK_API_KEY=${clerk_api_key.backend.secret}"
}
```

When you print such a value with an output, mark the output `sensitive`.

## Rotation

- `clerk_api_key`, `clerk_machine`: replace the resource
  (`terraform apply -replace=...`); the new secret flows through the graph.
- `clerk_oauth_application`: the same replace rotates the client secret.
- The instance secret key of the provider rotates in the Clerk dashboard;
  Terraform does not manage it.
