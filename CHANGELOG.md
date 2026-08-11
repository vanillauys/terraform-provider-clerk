# Changelog

## [Unreleased]

## [0.2.0] - 2026-08-11

- Add the `clerk_instance_settings` singleton resource. It manages the
  general instance settings and `allowed_origins` (readable, with drift
  detection); the other fields are write-only.
- Add the `clerk_instance_restrictions` singleton resource with full drift
  detection through the empty-PATCH read.
- Add the `clerk_instance_organization_settings` singleton resource with
  full drift detection through the empty-PATCH read.
- Add the nightly acceptance workflow. It runs when the
  `CLERK_SECRET_KEY_ACC` secret exists and skips cleanly when it does not.
- Make `make testacc` serial (`-p 1`): the instance singletons share one
  instance and parallel test packages race.

## [0.1.0] - 2026-08-11

- Add the `clerk` provider with `secret_key` and `api_url` configuration and
  support for aliased providers (one instance per provider block).
- Add the `clerk_jwt_template` resource.
- Add the `clerk_redirect_url` resource.
- Add the `clerk_allowlist_identifier` resource.
- Add the `clerk_blocklist_identifier` resource.
- Add the `clerk_domains` data source with the CNAME targets for DNS.
