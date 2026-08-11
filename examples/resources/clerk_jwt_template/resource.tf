resource "clerk_jwt_template" "api" {
  name     = "api"
  lifetime = 3600

  claims = jsonencode({
    role  = "{{user.public_metadata.role}}"
    email = "{{user.primary_email_address}}"
  })
}
