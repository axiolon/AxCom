## Description

Please include a summary of the changes and the related issue/ticket. Provide context, design decisions, and any testing details.

Fixes # (issue number)

## Type of Change

Please mark the options that apply:

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update (changes to docs or markdown files)
- [ ] Performance improvement

## How Has This Been Tested?

Describe the steps you took to verify your changes. Include details of your testing environment, and the tests you ran to see how your changes affect other areas of the code.

- **Unit tests run:** (e.g. `go test -v ./internal/core/cart/...`)
- **E2E tests run:** (e.g. `go test -tags e2e -v ./tests/e2e/...`)
- **Manual verification steps:** (explain how you tested this locally)

## Checklist

By submitting this PR, I confirm that:

- [ ] My code follows the code style and conventions of this project.
- [ ] I have run `golangci-lint run` locally and verified no new issues are reported.
- [ ] I have added tests that prove my fix is effective or that my feature works.
- [ ] Existing and new unit tests pass locally with my changes.
- [ ] I have updated the documentation to reflect my changes (if applicable).
- [ ] I have run `go run github.com/google/addlicense@latest` (or lefthook) to verify license headers are present on all new files.
- [ ] My commits follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.
