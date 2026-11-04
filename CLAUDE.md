# CLAUDE.md

Guidance for a coding agent that works in this repository. People read
[CONTRIBUTING.md](CONTRIBUTING.md); it holds the full rules, and this file
does not repeat them.

## What this is

The Terraform provider `vanillauys/clerk` for the Clerk Backend API. Go,
`terraform-plugin-framework`, one package per resource under
`internal/resources/`, one per data source under `internal/datasources/`,
and `clerk-sdk-go` behind `internal/clerkapi/`. The provider pins the
Backend API version that `clerk-sdk-go` pins.

## Commands

| Task | Command |
|------|---------|
| Build | `make build` |
| Unit tests and the schema snapshot | `make test` |
| Lint (must print `0 issues`) | `make lint` |
| Vulnerability scan | `make vuln` |
| Regenerate `docs/` | `make docs` |
| Regenerate the schema snapshot | `go test ./internal/provider -run TestSchemaSnapshot -update` |
| Acceptance tests | `CLERK_SECRET_KEY=$(op read op://wihan_dev/terraform-provider-clerk-dev/clerk_secret_key) make testacc` |

Set `CGO_ENABLED=0` on every direct `go` or `golangci-lint` command; the
Makefile sets it for its targets. Never point acceptance tests at a
production instance (`sk_live_...`).

## Rules that break the build or the release

- **A state from the previous minor release plans no change.** A new
  attribute on an existing resource is `Optional` without a `Default` in
  most cases. `TestAccUpgradeFromPrevious` (`internal/upgrade`) proves it.
  After each minor release, move its `previousRelease` constant.
- **Every optional attribute reverts when removed**, and an acceptance step
  proves it with an empty plan. Clerk ignores an absent update field: clear
  a string with `tfutil.ClearableString` and read it with `tfutil.NullIfEmpty`.
- **Fix a bug on every sibling resource**, with the copied test.
- **`docs/` is generated.** Edit `templates/`, `examples/`, or the schema
  `Description`, then run `make docs`. CI fails on a diff.
- **The schema snapshot guards the semver promise.** Regenerate
  `internal/provider/testdata/schema.json` only for an intended change.
- **`go.mod` stays tidy.** CI fails when `go mod tidy` changes it.
- **Sign every commit.** The `master` ruleset rejects an unsigned commit. Never pass
  `--no-gpg-sign` or `--no-verify`. Stage files by name.
- **`CHANGELOG.md` is the version record.** Add a line under
  `## [Unreleased]` for each user-visible change.
