terraform {
  required_providers {
    clerk = {
      source = "vanillauys/clerk"
    }
  }
}

# One provider block targets one Clerk instance.
# The secret key falls back to the CLERK_SECRET_KEY environment variable.
provider "clerk" {
  secret_key = var.clerk_secret_key_dev
}

# Use an alias for a second instance, for example production.
provider "clerk" {
  alias      = "prod"
  secret_key = var.clerk_secret_key_prod
}

resource "clerk_jwt_template" "api" {
  name     = "api"
  lifetime = 3600

  claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })
}

resource "clerk_jwt_template" "api_prod" {
  provider = clerk.prod

  name     = "api"
  lifetime = 3600

  claims = jsonencode({
    role = "{{user.public_metadata.role}}"
  })
}
