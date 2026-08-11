resource "clerk_api_key" "backend" {
  name    = "backend"
  subject = "user_2abcDEFghiJKLmnoPQRstuVWXyz"
  scopes  = ["read", "write"]

  # Optional: expire the key 90 days from the apply.
  seconds_until_expiration = 90 * 24 * 60 * 60
}

output "backend_api_key" {
  value     = clerk_api_key.backend.secret
  sensitive = true
}
