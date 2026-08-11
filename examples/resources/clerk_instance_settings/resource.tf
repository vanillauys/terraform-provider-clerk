resource "clerk_instance_settings" "this" {
  test_mode     = true
  support_email = "support@example.com"

  # Origins for native apps and cross-origin requests. The provider reads
  # this list back and detects drift.
  allowed_origins = [
    "capacitor://localhost",
  ]
}
