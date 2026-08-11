# CGO is off for every target: release builds already compile with
# CGO_ENABLED=0 (.goreleaser.yml) and nothing here needs cgo.
export CGO_ENABLED = 0

default: build

build: hooks
	go build ./...

test:
	go test ./... -count=1

# Acceptance tests run against a real Clerk instance and mutate its
# configuration. Point CLERK_SECRET_KEY at a development instance.
testacc:
	TF_ACC=1 go test ./internal/... -run 'TestAcc' -v -timeout 30m

lint:
	golangci-lint run

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name clerk

# Point git at the version-controlled hooks in .githooks/ so every clone gets
# the gitleaks pre-commit secret scan without a manual step. Idempotent; no-op
# outside a git working copy (e.g. CI source archives).
hooks:
	@git rev-parse --git-dir >/dev/null 2>&1 && git config core.hooksPath .githooks || true

.PHONY: default build test testacc lint docs hooks
