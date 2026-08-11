---
page_title: "Dev and prod instances"
subcategory: ""
description: |-
  Manage the development and the production instance of one application in one plan, with provider aliases.
---

# Dev and prod instances

A Clerk application has separate instances (development and production),
and each instance has its own secret key. One provider block targets one
instance. Use an alias for the second one:

```terraform
provider "clerk" {
  secret_key = var.clerk_secret_key_dev
}

provider "clerk" {
  alias      = "prod"
  secret_key = var.clerk_secret_key_prod
}
```

Resources without a `provider` argument go to the default (dev) provider.
Give the production copy the alias:

```terraform
resource "clerk_jwt_template" "api_dev" {
  name     = "api"
  lifetime = 3600
}

resource "clerk_jwt_template" "api_prod" {
  provider = clerk.prod

  name     = "api"
  lifetime = 3600
}
```

Repeat this pattern for every resource that both instances need. The two
copies stay independent: an apply against dev never touches prod.

## DNS for the production instance

A production instance needs CNAME records in DNS. The `clerk_domains`
data source returns them, so your DNS provider configuration can consume
them directly:

```terraform
data "clerk_domains" "prod" {
  provider = clerk.prod
}

locals {
  clerk_primary = one([
    for d in data.clerk_domains.prod.domains : d if !d.is_satellite
  ])

  # host => target, for example "clerk.example.com" => "frontend-api.clerk.services"
  clerk_dns = {
    for t in local.clerk_primary.cname_targets : t.host => t.value
  }
}

resource "cloudflare_dns_record" "clerk" {
  for_each = local.clerk_dns

  zone_id = var.zone_id
  name    = each.key
  type    = "CNAME"
  content = each.value
  ttl     = 1

  # Clerk records must stay DNS-only (not proxied).
  proxied = false
}
```

## Which resources differ per environment

Keep these points in mind when you mirror resources:

- `clerk_instance_settings`: `test_mode` and `development_origin` apply to
  development instances; `enhanced_email_deliverability` applies to
  production.
- `clerk_domain` (satellite domains) and the DNS data normally matter on
  production only.
- API keys and machines (`clerk_api_key`, `clerk_machine`) mint separate
  secrets per instance by design.
