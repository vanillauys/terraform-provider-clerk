---
page_title: "Getting started"
subcategory: ""
description: |-
  Configure the provider and apply a first JWT template, redirect URL, and sign-up restriction.
---

# Getting started

## Get a secret key

The provider talks to the [Clerk Backend API](https://clerk.com/docs/reference/backend-api)
with an instance secret key. Get it from the Clerk dashboard under
**API keys**: `sk_test_...` for a development instance, `sk_live_...` for
production.

Start against a development instance. The provider mutates real instance
configuration, and a development instance is disposable.

## Configure the provider

```terraform
terraform {
  required_providers {
    clerk = {
      source  = "vanillauys/clerk"
      version = "~> 0.7"
    }
  }
}

provider "clerk" {
  secret_key = var.clerk_secret_key
}
```

The provider also reads `CLERK_SECRET_KEY` from the environment, so you
can keep the key out of your variables:

```sh
CLERK_SECRET_KEY=sk_test_... terraform plan
```

## A first apply

```terraform
# The shape of the session tokens for your API.
resource "clerk_jwt_template" "api" {
  name     = "api"
  lifetime = 3600

  claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })
}

# Allow the SSO callback of your app.
resource "clerk_redirect_url" "app" {
  url = "https://app.example.com/sso-callback"
}

# Block disposable email addresses at sign-up.
resource "clerk_instance_restrictions" "this" {
  block_disposable_email_domains = true
}
```

Run `terraform apply`. The `clerk_instance_restrictions` resource is a
singleton: it adopts the instance on create, and destroy only removes it
from state.

## Rate limits

The Backend API rate-limits requests per secret key. A very large apply
can hit the limit; the provider surfaces the API error as-is. Split very
large changes, or apply them again after a short wait.

## Next steps

- [Dev and prod instances](dev-and-prod-instances) — one plan, two
  instances, provider aliases.
- [Adopting an existing instance](adopting-an-existing-instance) — bring
  a configured instance under Terraform without recreation.
- [Secrets and sensitive values](secrets) — what lands in state and how
  to treat it.
