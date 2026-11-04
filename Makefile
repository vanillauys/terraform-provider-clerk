# CGO is off for every target: release builds already compile with
# CGO_ENABLED=0 (.goreleaser.yml) and nothing here needs cgo.
export CGO_ENABLED = 0

# When COVERDIR is set, every test binary writes binary coverage data to
# that directory, and `go tool covdata textfmt -i=$COVERDIR -o=coverage.out`
# merges the files into one profile. acceptance.yml sets it for the
# SonarQube Cloud analysis. The path must be absolute: each test binary
# runs in its own package directory.
ifdef COVERDIR
COVERFLAGS = -cover -coverpkg=./internal/... -args -test.gocoverdir=$(COVERDIR)
endif

default: build

build: hooks
	go build ./...

test:
	go test ./... -count=1 $(COVERFLAGS)

# Acceptance tests run against a real Clerk instance and mutate its
# configuration. Point CLERK_SECRET_KEY at a development instance.
# -p 1 is load-bearing: the instance singletons (settings, restrictions,
# organization settings) share one instance, and parallel packages race.
# -count=1 is load-bearing too: without it go test reuses cached passes and
# the suite reports green without touching the API.
testacc:
	TF_ACC=1 go test ./internal/... -run 'TestAcc' -v -timeout 30m -p 1 -count=1 $(COVERFLAGS)

lint:
	golangci-lint run

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# The Changelog guide is CHANGELOG.md behind a registry front matter. It is
# written after tfplugindocs because generate empties docs/guides/ first, and
# no template function includes a file as markdown.
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name clerk
	printf -- '---\npage_title: "Changelog"\nsubcategory: ""\ndescription: |-\n  Every release of the provider, newest first. Generated from CHANGELOG.md.\n---\n\n' > docs/guides/changelog.md
	cat CHANGELOG.md >> docs/guides/changelog.md

# Point git at the version-controlled hooks in .githooks/ so every clone gets
# the gitleaks pre-commit secret scan without a manual step. Idempotent; no-op
# outside a git working copy (e.g. CI source archives).
hooks:
	@git rev-parse --git-dir >/dev/null 2>&1 && git config core.hooksPath .githooks || true

.PHONY: default build test testacc lint vuln docs hooks
