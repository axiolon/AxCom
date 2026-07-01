# Contributing to AxCom

Thank you for your interest in contributing to AxCom! We welcome contributions from the community to help make this high-performance e-commerce engine better.

This project is licensed under the **Apache License 2.0** (see [LICENSE](LICENSE)). By submitting a contribution, you agree that your work will be licensed under the same terms.

---

## How Can I Contribute?

### 1. Reporting Bugs

Before opening a bug report, please check existing issues. If the bug is new:

- Go to [Issues](https://github.com/axiolon/axcom/issues) and select the **Bug Report** template.
- Provide all requested information, including environment details, reproduction steps, and logs.

### 2. Suggesting Enhancements

For feature requests or design proposals:

- Check existing issues and Discussions first.
- If it hasn't been proposed yet, open a **Feature Request** issue to start a discussion with the maintainers.

### 3. Submitting Code

We love pull requests! To submit a change:

1. **Fork** the repository and create your feature branch from `main` (e.g., `feat/add-discount-rules` or `fix/jwt-expiration`).
2. Write tests for your changes.
3. Make sure all tests pass locally.
4. Run code quality checks (see the **Development Standards** section below).
5. Ensure license headers are present on all new source files.
6. Open a Pull Request against the `main` branch of the upstream repository.

---

## Development Standards

### Commit Messages

We enforce [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for clear, structured release notes. Your commit messages should follow this format:

```text
<type>(<scope>): <description>

[optional body]
```

Common types:

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation-only changes
- `style`: Changes that do not affect the meaning of the code (formatting, missing semi-colons, etc.)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Build processes, tooling, or auxiliary library updates

### Code Quality (Linting & Testing)

Before submitting a PR, make sure your Go code meets standard checks:

- **Linting:** Run `golangci-lint run` locally. The CI pipeline will reject builds with linter warnings.
- **Tests:** Run backend tests locally with race detection:
  ```bash
  go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
  ```

### License Compliance

Every source file (`.go`, `.ts`, `.yml`, etc.) must carry an Apache 2.0 license header. We use `lefthook` to check this on pre-commit. You can check this manually or run:

```bash
go run github.com/google/addlicense@latest -check .
```

To automatically apply the license header:

```bash
go run github.com/google/addlicense@latest -c "Axiolon Labs" -l apache -s=only [modified_files]
```

Alternatively, you can run the helper scripts provided in the `scripts/` directory to check or apply license headers across the repository:

**On Linux/macOS:**
```bash
./scripts/add-headers.sh --check  # to check
./scripts/add-headers.sh          # to apply
```

**On Windows (PowerShell):**
```powershell
.\scripts\add-headers.ps1 --check # to check
.\scripts\add-headers.ps1         # to apply
```

---

## Contributor Onboarding Guide

For a step-by-step developer environment setup (Go, MongoDB, Redis, configuration files), please check out the official **[Contributor Onboarding Guide](docs/contributing/onboarding.md)** in our documentation directory.

---

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) in all community interactions.
