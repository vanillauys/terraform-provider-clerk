data "clerk_jwks" "this" {}

output "signing_key_ids" {
  value = [for k in data.clerk_jwks.this.keys : k.kid]
}
