# Security policy

## Report a vulnerability

Report a vulnerability through
[GitHub private vulnerability reporting](https://github.com/vanillauys/terraform-provider-clerk/security/advisories/new).
Do not open a public issue for it. You get a first reply within seven days.

## Scope

The provider sends the Clerk secret key to the Clerk Backend API over
HTTPS, and it keeps API key secrets, machine secret keys, and OAuth client
secrets in the Terraform state. In scope:

- The provider binary and its API client (`internal/`).
- The release pipeline: the GoReleaser build, the GPG signature, the build
  provenance attestation, and the pinned actions in `.github/workflows/`.
- The generated documentation, if it tells a user to do something unsafe.

Out of scope:

- Clerk itself. Report a Clerk problem to
  [Clerk](https://clerk.com/docs/security/vulnerability-disclosure-policy).
- The Terraform state. The state holds attribute values in cleartext by
  design. The
  [Secrets guide](https://registry.terraform.io/providers/vanillauys/clerk/latest/docs/guides/secrets)
  explains how to protect it.
- Terraform and OpenTofu.

## Supported versions

The latest minor release gets fixes. Move to it before you report.

## What each release carries

GoReleaser builds each release in GitHub Actions and signs the checksum
file with the GPG key `750EE4482941313E`, the key that the Terraform
registry verifies. Each release archive carries a GitHub build provenance
attestation. Every action in the workflows is pinned to a commit SHA.
`govulncheck` runs on every pull request and fails it on a reachable
vulnerability.
