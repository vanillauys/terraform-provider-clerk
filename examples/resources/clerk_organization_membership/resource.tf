resource "clerk_organization_membership" "wihan" {
  organization_id = clerk_organization.acme.id
  user_id         = "user_2abcDEFghiJKLmnoPQRstuVWXyz"
  role            = "org:admin"
}
