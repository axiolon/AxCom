# Copyright 2026 Axiolon Labs
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT := $(shell git rev-parse --show-toplevel 2>/dev/null || echo ..)

.PHONY: license-add license-check dep-license-check

# Add license headers to all files in the repository
license-add:
	go run github.com/google/addlicense@v1.1.1 \
		-c "Axiolon Labs" \
		-l apache \
		-s=only \
		-ignore "**/node_modules/**" \
		-ignore "**/dist/**" \
		-ignore "**/.next/**" \
		-ignore "**/.git/**" \
		-ignore "**/vendor/**" \
		-ignore "**/.idea/**" \
		-ignore "**/.vscode/**" \
		-ignore "**/*.json" \
		-ignore "**/*.md" \
		-ignore "**/LICENSE" \
		-ignore "**/NOTICE" \
		-ignore "**/Dockerfile" \
		$(REPO_ROOT)

# Check that all files in the repository have license headers
license-check:
	go run github.com/google/addlicense@v1.1.1 \
		-check \
		-c "Axiolon Labs" \
		-l apache \
		-s=only \
		-ignore "**/node_modules/**" \
		-ignore "**/dist/**" \
		-ignore "**/.next/**" \
		-ignore "**/.git/**" \
		-ignore "**/vendor/**" \
		-ignore "**/.idea/**" \
		-ignore "**/.vscode/**" \
		-ignore "**/*.json" \
		-ignore "**/*.md" \
		-ignore "**/LICENSE" \
		-ignore "**/NOTICE" \
		-ignore "**/Dockerfile" \
		$(REPO_ROOT)

# Check Go dependencies for allowed licenses using the allowlist file
dep-license-check:
	@echo "Checking Go dependency licenses..."
	go run github.com/google/go-licenses@latest check --allowed_licenses=Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC ./...
