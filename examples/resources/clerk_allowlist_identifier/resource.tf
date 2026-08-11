# Allow one address.
resource "clerk_allowlist_identifier" "founder" {
  identifier = "founder@example.com"
}

# Allow a whole domain.
resource "clerk_allowlist_identifier" "company" {
  identifier = "*@example.com"
}
