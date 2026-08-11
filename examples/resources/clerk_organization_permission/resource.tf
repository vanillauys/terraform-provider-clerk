resource "clerk_organization_permission" "invoices_read" {
  name        = "Read invoices"
  key         = "org:invoices:read"
  description = "Read access to the invoices of the organization."
}
