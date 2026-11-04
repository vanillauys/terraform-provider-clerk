# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.0] - 2026-11-05

### Added

- `clerk_jwt_template`: `signing_key_wo` and `signing_key_wo_version`, the
  write-only form of `signing_key`. Terraform keeps the key out of the plan
  and the state. It needs Terraform 1.11 or later. Change
  `signing_key_wo_version` to send a new key.
- The `clerk_organizations` data source lists the organizations of the
  instance. The optional `query` matches the id, the name, or the slug.

### Changed

- clerk-sdk-go moves to v2.8.0. It replaces go-jose v3 with go-jose v4.1.6
  for GHSA-mgcc-cww5-fp2q. go-jose v3 also had GHSA-4w3g-jfj2-cw2q and
  GHSA-vvm8-gf78-28g4, with no fixed v3 release. The provider calls none of
  the affected code. The Backend API version stays `2026-05-12`.
- Each optional attribute with a fixed Clerk default now has that default
  in the schema. If you remove the attribute from the configuration, it
  reverts to the default. Before, Clerk kept the last value. The defaults
  are the values that Clerk sets on create:
  - `clerk_jwt_template`: `lifetime` 60, `allowed_clock_skew` 5,
    `custom_signing_key` false, `signing_algorithm` `RS256`, `claims` `{}`.
  - `clerk_machine`: `default_token_ttl` 3600, `scoped_machines` empty.
  - `clerk_api_key`: `claims` `{}`, `scopes` empty.
  - `clerk_oauth_application`: `public` false, `consent_screen_enabled`
    true.
  - `clerk_organization`: `admin_delete_enabled` true, `public_metadata`
    and `private_metadata` `{}`.
  - `clerk_organization_role`: `permissions` empty.
  - `clerk_saml_connection`: `active`, `allow_subdomains`,
    `allow_idp_initiated` and `force_authn` false, `sync_user_attributes`
    true.

  **Upgrade note:** if a resource holds a value that is not the default, and
  the configuration does not set the attribute, the next plan changes the
  value to the default. This can occur after an import, or after a change in
  the dashboard. Before you apply, set the attribute in the configuration
  to keep the value.
- `clerk_oauth_application`: `scopes` defaults to `"email profile"`, which
  Clerk stores as its default `email offline_access profile`. Before, a
  removed `scopes` kept the last value on the server.
- **BREAKING:** `clerk_instance_organization_settings.domains_enrollment_modes`
  is now a set. Clerk returns the modes sorted, so a list in another order
  failed the apply with "Provider produced inconsistent result". An existing
  state loads with no change.
- `clerk_api_key`: an update sends `seconds_until_expiration` only when the
  value changes. Before, an unrelated change, for example to the
  description, restarted the lifetime.
- `clerk_organization` `slug`, `max_allowed_memberships` and `created_by`,
  and the `clerk_saml_connection` `idp_` fields, have no fixed default:
  Clerk derives them. Their descriptions say what occurs when you remove
  them.
- `clerk_organization_domain`: `enrollment_mode` defaults to
  `manual_invitation`. Clerk rejects a create without a mode, so a
  configuration without it failed before. If you remove the attribute, it
  reverts to the default.
- The indirect Go modules with known vulnerabilities move to fixed
  versions: golang.org/x/net v0.62.0, golang.org/x/text v0.43.0,
  google.golang.org/grpc v1.85.0 and github.com/go-jose/go-jose/v3 v3.0.5.

- The Go toolchain moves to 1.26.10. `govulncheck` reported GO-2026-6809 in
  the Go 1.26.9 `crypto/x509` package.
- CI runs the acceptance suite on each pull request and each push to
  `master`, SonarQube Cloud analysis with the merged coverage, OpenSSF
  Scorecard, and CodeQL. The nightly run tests Terraform 1.5.7, the latest
  Terraform, and OpenTofu 1.12.9.
- Each release archive carries a GitHub build provenance attestation. Verify
  an archive with `gh attestation verify <archive> --owner vanillauys`.

### Fixed

- If you remove an optional attribute from the configuration, Clerk now
  clears the value. Before, the apply failed with "Provider produced
  inconsistent result". This applies to:
  - `description` on `clerk_organization_permission`,
    `clerk_organization_role`, and `clerk_api_key`.
  - `proxy_url` on `clerk_domain`.
  - `organization_id` on `clerk_saml_connection`.
- `clerk_api_key`: if you remove all `scopes`, Clerk now clears them. The
  SDK dropped an empty scopes list from the update request.
- `clerk_saml_connection`: a PEM `idp_certificate` no longer fails the apply
  with "Provider produced inconsistent result". Clerk stores the bare
  base64 form, and the provider now keeps the configured form.
- `clerk_saml_connection`: if the update after the create fails, the error
  now shows the id of the connection that Clerk created.
- The provider retries a request that Clerk rate-limits (429), up to five
  times, and waits as long as `Retry-After` asks (at most 30 seconds). Before,
  a large apply failed with "too_many_requests". The client also drops the
  SDK's 5-second timeout for the whole call; the Terraform operation and a
  30-second header timeout for each attempt now bound it. A fuzz test
  found that a very large `Retry-After` overflowed the wait; the provider
  now caps the value before it computes the wait.
- `clerk_organization_domain`: the read pages through all the domains of the
  organization. Before, it read only the first page of 10, so a domain on a
  later page dropped from the state.
- `clerk_organization` and `clerk_saml_connection`: if the update after the
  create fails, the state keeps the created record. Before, the record
  leaked and the next apply made a duplicate.
- `clerk_api_key` and `clerk_machine`: if the record goes away between the
  read and the secret read, the resource leaves the state. Before, the
  refresh failed.
- `clerk_organization_membership`: the read filters by `user_id`. Before,
  the read searched only the first page of 10 members. In a larger
  organization, a membership could drop from the state.

## [0.7.0] - 2026-08-11

- Add the four guides: getting started, dev and prod instances, adoption
  of an existing instance with import, and secrets.
- Rework the index page and the README around the full resource surface.

## [0.6.0] - 2026-08-11

- Add the `clerk_oauth_application` resource. `client_secret` comes from
  the create response and stays in state; `scopes` is write-only because
  Clerk canonicalizes the value server-side (order, plus a forced
  `offline_access`).
- Add the `clerk_saml_connection` resource. The identity-provider key is
  `provider_key` (`provider` is a reserved attribute name in Terraform).

## [0.5.0] - 2026-08-11

- Add the `clerk_organization` resource with metadata support.
- Add the `clerk_organization_permission` and `clerk_organization_role`
  resources. A role's permission set updates in place as a whole list.
- Add the `clerk_organization_domain` resource (import id
  `organization_id/domain_id`). Needs `domains_enabled = true` on the
  instance organization settings.
- Add the `clerk_organization_membership` resource (import id
  `organization_id/user_id`).
- Add the `clerk_organization` data source with lookup by id or slug.

## [0.4.0] - 2026-08-11

- Add the `clerk_domain` resource for satellite domains. The Backend API
  cannot create a primary domain; a primary-domain change stays in the
  dashboard.
- Add the `clerk_webhook` singleton resource: create enables the Svix
  integration, destroy disables it. Write-only; the API has no read for it.
- Add the `clerk_jwks` data source with the key ids and algorithms of the
  instance.

## [0.3.0] - 2026-08-11

- Add the `clerk_api_key` resource. The secret stays in state (sensitive)
  and Read keeps it fresh through the secret endpoint; a revoked or
  expired key leaves state so the next apply replaces it.
- Add the `clerk_machine` resource with `scoped_machines` reconcile
  through the machine-scopes API.
- BREAKING: `clerk_instance_settings.allowed_origins` is now a set. As a
  list it produced phantom refresh diffs because Clerk returns the origins
  in arbitrary order.
- Make `make testacc` bypass the Go test cache (`-count=1`): a cached pass
  reported green without any API call.

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
