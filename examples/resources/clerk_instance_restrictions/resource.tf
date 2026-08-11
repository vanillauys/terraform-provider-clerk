resource "clerk_instance_restrictions" "this" {
  blocklist                      = true
  block_email_subaddresses       = true
  block_disposable_email_domains = true
}
