# Lookup by slug (the id works too).
data "clerk_organization" "acme" {
  lookup = "acme"
}

output "acme_org_id" {
  value = data.clerk_organization.acme.id
}
