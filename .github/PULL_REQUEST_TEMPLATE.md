## What this changes

<!-- One paragraph. Name the resource or the area, and the Clerk behavior that drives the change. -->

## Checklist for every pull request

- [ ] Every commit is signed and shows **Verified** on GitHub ([how to sign](https://github.com/vanillauys/terraform-provider-clerk/blob/master/CONTRIBUTING.md#sign-your-commits))
- [ ] `CHANGELOG.md`: a line under `## [Unreleased]`
- [ ] `make test`, `make lint` (0 issues), and `make docs` pass

## Checklist for a resource or data source change

- [ ] `internal/resources/<name>/` or `internal/datasources/<name>/`: schema, CRUD, and `ImportState`
- [ ] `*_acc_test.go`: a step that drops each optional attribute and asserts an empty plan; import with `ImportStateVerify`
- [ ] `internal/provider/provider.go`: the registration
- [ ] `internal/provider/testdata/schema.json`: the regenerated schema snapshot, for an intended schema change only
- [ ] `examples/resources/clerk_<name>/`: `resource.tf` and `import.sh`
- [ ] `make docs`: the generated page is committed
- [ ] `templates/index.md.tmpl` and `README.md`: the two files nothing in CI checks

## Verification

<!-- Paste the unedited output of the commands you ran: unit tests, lint, and the acceptance tests for the packages you touched. -->
