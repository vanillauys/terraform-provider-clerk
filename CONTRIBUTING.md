# Contributing

Thank you for your help. This guide tells you how to report a problem, how
to set up the project, and how to send a pull request that can merge.

> [!IMPORTANT]
> **Every commit must be signed.** GitHub must show **Verified** on each
> commit of your pull request. The `master` ruleset rejects an unsigned
> commit. Set up signing before your first commit: see
> [Sign your commits](#sign-your-commits).

## Contents

- [Report a bug or ask for a feature](#report-a-bug-or-ask-for-a-feature)
- [What you need](#what-you-need)
- [Send a pull request, step by step](#send-a-pull-request-step-by-step)
- [Sign your commits](#sign-your-commits)
- [What CI checks](#what-ci-checks)
- [Build, test, and lint](#build-test-and-lint)
- [Engineering rules](#engineering-rules)

## Report a bug or ask for a feature

- **Bug:** open a
  [bug report](https://github.com/vanillauys/terraform-provider-clerk/issues/new?template=bug_report.yml).
  Give the provider version, the Terraform version, the Clerk instance type
  (development or production), the smallest configuration that shows the
  problem, and the error. Remove every secret from the output first.
- **Feature:** open a
  [feature request](https://github.com/vanillauys/terraform-provider-clerk/issues/new?template=feature_request.yml)
  before you write code for a new resource or a new attribute. Name the
  Backend API endpoint or the dashboard setting that you want to manage.

A small fix (a typo, a wrong description, a one-line bug fix) does not need
an issue first.

## What you need

| Tool | Version | For |
|------|---------|-----|
| [Go](https://go.dev/dl/) | the `go` line in [`go.mod`](go.mod) | build and unit tests |
| [Terraform](https://developer.hashicorp.com/terraform/install) | a current release | acceptance tests |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | the `version:` in [`.github/workflows/test.yml`](.github/workflows/test.yml) | `make lint` |
| [gitleaks](https://github.com/gitleaks/gitleaks#installing) | 8.19 or later | the pre-commit hook |
| A Clerk development instance | - | acceptance tests |
| GNU Make | any | the targets below |

The unit tests, the linter, and the docs need only Go and golangci-lint. The
acceptance tests need the secret key of a Clerk development instance.

## Send a pull request, step by step

1. **Fork** the repository on GitHub, then clone your fork:

   ```sh
   git clone git@github.com:<you>/terraform-provider-clerk.git
   cd terraform-provider-clerk
   git remote add upstream https://github.com/vanillauys/terraform-provider-clerk.git
   ```

2. **Set up commit signing** for this clone. Follow
   [Sign your commits](#sign-your-commits), then check it:

   ```sh
   git commit --allow-empty -m "test: signing" && git log --show-signature -1
   git reset --hard HEAD~1
   ```

3. **Enable the git hooks.** The pre-commit hook scans the staged changes
   for secrets with gitleaks:

   ```sh
   make hooks
   ```

4. **Make a branch** from the latest `master`. Use a short name with a
   type prefix, for example `fix/jwt-template-lifetime` or
   `feat/organization-invitation`:

   ```sh
   git fetch upstream
   git switch -c fix/<short-name> upstream/master
   ```

5. **Make the change.** Keep one topic in one pull request. For a resource
   or a data source, the
   [pull request template](.github/PULL_REQUEST_TEMPLATE.md) lists every
   file that the change touches. Read
   [Engineering rules](#engineering-rules) before you change a schema.

6. **Test the change locally:**

   ```sh
   make test   # unit tests and the schema snapshot
   make lint   # must print "0 issues"
   make docs   # regenerate docs/ from the schema, templates/, and examples/
   ```

   For a change to a resource, run its acceptance tests on a development
   instance too. See [Acceptance tests](#acceptance-tests).

7. **Add a line to `CHANGELOG.md`** under `## [Unreleased]`, in the
   `Added`, `Changed`, or `Fixed` section. Write what the user sees, and
   the issue number.

8. **Commit** with a [Conventional Commits](https://www.conventionalcommits.org/)
   message: `feat:`, `fix:`, `docs:`, `test:`, `ci:`, or `chore:`. Stage
   the files by name; do not stage files that you did not change.

   ```sh
   git add internal/resources/jwttemplate/resource.go CHANGELOG.md
   git commit -m "fix: revert the JWT template lifetime when it is removed (#123)"
   ```

9. **Push** the branch to your fork and **open a pull request** against
   `master`. Fill in the template. Paste the output of the commands that
   you ran into **Verification**.

10. **Wait for the checks.** GitHub holds the workflow runs of a pull
    request from a fork until a maintainer approves them. The checks start
    after that approval. See [What CI checks](#what-ci-checks).

11. **Answer the review.** Push more commits to the same branch. Do not
    force-push after a review starts, so the reviewer can see what changed.
    If `master` moves, merge it into your branch.

The maintainer merges with **Squash and merge**. The squashed commit keeps
you as the author.

## Sign your commits

GitHub accepts an SSH signature or a GPG signature. SSH is the shorter
setup if you already push with an SSH key.

### With an SSH key

1. Tell git to sign with your SSH key:

   ```sh
   git config --global gpg.format ssh
   git config --global user.signingkey ~/.ssh/id_ed25519.pub
   git config --global commit.gpgsign true
   ```

2. Add the same public key to GitHub a second time, with the key type
   **Signing Key**: see
   [Adding a new SSH key to your GitHub account](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account).
   An **Authentication Key** does not verify commits.

3. To check signatures locally, add your key to an allowed signers file:

   ```sh
   echo "$(git config user.email) $(cat ~/.ssh/id_ed25519.pub)" >> ~/.ssh/allowed_signers
   git config --global gpg.ssh.allowedSignersFile ~/.ssh/allowed_signers
   ```

### With a GPG key

1. [Generate a GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
   and [add it to your GitHub account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account).
2. [Tell git about the key](https://docs.github.com/en/authentication/managing-commit-signature-verification/telling-git-about-your-signing-key):

   ```sh
   git config --global user.signingkey <KEY-ID>
   git config --global commit.gpgsign true
   ```

### Check and fix

- The email of the commit must be a verified email on your GitHub account.
  Else GitHub shows **Unverified**.
- `git log --show-signature` shows the signature of each commit locally.
  On GitHub, each commit of the pull request must show **Verified**.
- If a commit of your branch is not signed, sign every commit of the branch
  again and push:

  ```sh
  git rebase --exec 'git commit --amend --no-edit -S' upstream/master
  git push --force-with-lease
  ```

  This is the one case where a force-push to your branch is correct.

GitHub's guide
[About commit signature verification](https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification)
has the full detail, and
[Troubleshooting commit signature verification](https://docs.github.com/en/authentication/troubleshooting-commit-signature-verification)
explains an **Unverified** badge.

The project has no CLA and needs no DCO `Signed-off-by` line. The signature
proves who made the commit.

## What CI checks

Every pull request runs these checks:

| Check | What it does |
|-------|--------------|
| `unit` | `go test -race` on every package, including the schema snapshot |
| `lint` | golangci-lint with the repository configuration |
| `tidy` | `go mod tidy` makes no change |
| `docs` | `make docs` makes no change to `docs/` |
| `vuln` | `govulncheck` finds no reachable vulnerability |
| `gitleaks` | scans the tree and every commit of the pull request for secrets |
| `acceptance` | the acceptance suite against the Clerk development instance of the project |
| `sonar` | SonarQube Cloud analysis, with the coverage of the unit tests and the acceptance suite |
| `CodeQL` | GitHub code scanning (default setup) |

A pull request from a fork gets no secrets, so `acceptance` and `sonar`
skip it. The maintainer runs the suite for such a pull request before the
merge. The `nightly` workflow runs the suite each night with Terraform
1.5.7, the latest Terraform, and OpenTofu 1.12.9. It also fuzzes each
`Fuzz` target for a minute. All acceptance runs share
one concurrency group, because they share one instance. The `scorecard`
workflow audits the repository each week.

## Build, test, and lint

The Makefile exports `CGO_ENABLED=0` for each target. Release builds are
already cgo-free, and nothing in the tree needs cgo. If you call `go` or
`golangci-lint` directly on a machine without a C compiler, set
`CGO_ENABLED=0` yourself.

- `make build`: build the provider binary. It also runs `make hooks`.
- `make test`: run the unit tests.
- `make lint`: run golangci-lint. The configuration is in `.golangci.yml`.
  `max-same-issues` is 0, so one run shows every finding.
- `make vuln`: run `govulncheck` on every package.
- `make docs`: regenerate the registry docs, and the Changelog guide from
  `CHANGELOG.md`. Edit `templates/` and `examples/`, never `docs/`. CI fails
  when the regenerated docs differ from the committed docs.
- `go test ./internal/provider -run TestSchemaSnapshot -update`: regenerate
  `internal/provider/testdata/schema.json`, the pinned provider schema. The
  unit tests fail when the schema differs from the snapshot. Review the diff
  of that file in the pull request: a removed attribute, a type change, or an
  optional attribute that became required is a breaking change and needs a
  major version.

### Acceptance tests

The acceptance tests create, change, and delete real records on a Clerk
instance. **Never point them at a production instance.** Use the secret key
of a development instance (`sk_test_...`):

```sh
CLERK_SECRET_KEY=sk_test_... make testacc
```

`make testacc` runs the packages one at a time (`-p 1`). The instance
singletons (settings, restrictions, organization settings) share one
instance, and parallel packages change the same state.

### Git hooks

`make hooks` points git at `.githooks/`. The pre-commit hook scans the
staged changes for secrets with
[gitleaks](https://github.com/gitleaks/gitleaks). The allowlist is in
`.gitleaks.toml`. For a confirmed false positive, add a `gitleaks:allow`
comment on the line or extend the allowlist. Do not use `--no-verify`.

### Fuzz tests

A function that parses input from outside the provider (a response header,
a value that Clerk normalizes) gets a native Go fuzz test with a seed
corpus. `make test` runs the seeds. To explore new inputs locally:

```sh
go test ./internal/clerkapi -run='^$' -fuzz='^FuzzRetryAfter$' -fuzztime=60s
```

A failing input lands in `testdata/fuzz/` of the package. Commit it with
the fix: it then stays in the seed corpus.

Add each new target to the `fuzz` job of `.github/workflows/nightly.yml`.

### Test layout

Acceptance tests live in `*_acc_test.go` files with `package <pkg>_test`, the
external test package. The `internal/acctest` package imports the provider,
and the provider imports each resource package. An internal test file that
imports `acctest` therefore creates an import cycle. Unit tests stay in the
internal package.

## Engineering rules

### A state from the previous release plans no change

A minor release must load a configuration and a state from the previous
minor release with an empty plan. A new attribute on an existing resource is
therefore `Optional` without a `Default` in most cases: an old state holds
null for it, and a default turns that null into a diff. The upgrade test
`TestAccUpgradeFromPrevious` in `internal/upgrade` applies with the released
provider version in `previousRelease`, then plans with this build, and
expects an empty plan. After each minor release, move `previousRelease` to
the new version. When you add a resource with a server-side read, add it to
the configuration of that test in the next minor release.

### Optional attributes must revert

When a user removes an optional attribute from the configuration, the
attribute must revert. The acceptance test must prove it with a step that
drops the attribute and asserts `plancheck.ExpectEmptyPlan()`. There are two
shapes, and the difference matters:

- **`Optional` without `Computed` or `Default`** reverts to **null**. The
  server must clear the value. Assert this with a direct API read, not only
  with the Terraform state.
- **`Optional` with `Computed` and a `Default`** reverts to **the default**,
  never to null. Assert the default value, and assert that the server holds
  it.

Clerk ignores a field that is absent from an update request. To clear an
optional string, send `""`: use `tfutil.ClearableString` in the update and
`tfutil.NullIfEmpty` in the read.

Find the default with a create that leaves the attribute out, then read the
record. If Clerk derives the value from another field or from an instance
setting, the attribute has no `Default`. Its `Description` must then say
what occurs when a user removes it.

The instance singletons (`clerk_instance_settings`,
`clerk_instance_restrictions`, `clerk_instance_organization_settings`) are
the exception. An attribute that you remove from a singleton stops under
Terraform management, and Clerk keeps its value.

### The sweep rule

A defect on one resource is usually also on its sibling resources. Fix it
on **all** of them, and copy the acceptance step that proves the fix. A fix
without the copied test does not prove the fix on the sibling resources.

### `docs/` is generated

Do not edit a file in `docs/`. Edit `templates/`, `examples/`, or the
`Description` of the schema attribute, then run `make docs`.

### The schema snapshot guards the semver promise

`internal/provider/testdata/schema.json` pins the provider schema. Regenerate
it only for an intended schema change, with
`go test ./internal/provider -run TestSchemaSnapshot -update`. A breaking
change in that diff needs a major version.

### Every user-visible change goes in the CHANGELOG

Add a line under `## [Unreleased]` in `CHANGELOG.md` for each change that a
user can see. `make docs` copies the file into the Changelog guide.
