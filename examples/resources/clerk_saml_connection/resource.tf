resource "clerk_saml_connection" "okta" {
  name         = "Okta"
  domain       = "example.com"
  provider_key = "saml_okta"

  idp_metadata_url = "https://example.okta.com/app/abc123/sso/saml/metadata"

  active           = true
  allow_subdomains = true
}

# Configure these two values at the identity provider.
output "saml_acs_url" {
  value = clerk_saml_connection.okta.acs_url
}

output "saml_sp_entity_id" {
  value = clerk_saml_connection.okta.sp_entity_id
}
