resource "clerk_organization_role" "billing_admin" {
  name        = "Billing admin"
  key         = "org:billing_admin"
  description = "Full access to billing."

  permissions = [
    clerk_organization_permission.invoices_read.id,
  ]
}
