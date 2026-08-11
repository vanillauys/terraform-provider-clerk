resource "clerk_organization" "acme" {
  name                    = "Acme"
  slug                    = "acme"
  max_allowed_memberships = 25

  public_metadata = jsonencode({
    tier = "enterprise"
  })
}
