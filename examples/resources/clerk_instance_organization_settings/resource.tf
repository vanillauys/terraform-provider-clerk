resource "clerk_instance_organization_settings" "this" {
  enabled                 = true
  max_allowed_memberships = 5
  admin_delete_enabled    = true
}
