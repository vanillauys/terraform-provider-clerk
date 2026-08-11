resource "clerk_oauth_application" "partner_portal" {
  name         = "Partner portal"
  callback_url = "https://portal.example.com/oauth/callback"
  scopes       = "profile email"
}

output "partner_portal_client_secret" {
  value     = clerk_oauth_application.partner_portal.client_secret
  sensitive = true
}
