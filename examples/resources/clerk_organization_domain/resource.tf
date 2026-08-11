# Needs domains_enabled = true on clerk_instance_organization_settings.
resource "clerk_organization_domain" "acme" {
  organization_id = clerk_organization.acme.id
  name            = "acme.com"
  enrollment_mode = "automatic_invitation"
}
