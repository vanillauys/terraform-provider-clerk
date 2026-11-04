# Every organization of the instance.
data "clerk_organizations" "all" {}

# Only the organizations whose id, name, or slug match the query.
data "clerk_organizations" "acme" {
  query = "acme"
}

output "organization_slugs" {
  value = [for o in data.clerk_organizations.all.organizations : o.slug]
}
