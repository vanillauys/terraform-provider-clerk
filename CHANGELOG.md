# Changelog

## [Unreleased]

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
