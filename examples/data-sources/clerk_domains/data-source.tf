data "clerk_domains" "all" {}

# The CNAME records that the primary domain needs in DNS, keyed by host.
# Feed this map into your DNS provider (for example Cloudflare) to keep the
# Clerk records in code.
locals {
  primary_domain = one([
    for d in data.clerk_domains.all.domains : d if !d.is_satellite
  ])

  clerk_dns = {
    for t in local.primary_domain.cname_targets : t.host => t.value
  }
}
