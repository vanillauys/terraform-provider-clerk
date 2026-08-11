resource "clerk_domain" "satellite" {
  name      = "satellite.example.com"
  proxy_url = "https://satellite.example.com/__clerk"
}
