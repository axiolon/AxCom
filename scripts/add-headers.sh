#!/bin/bash
# Copyright 2026 Axiolon Labs
# SPDX-License-Identifier: Apache-2.0

set -e

# Get repository root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || dirname "$(dirname "$0")")

# Check if checking or adding
CHECK_MODE=false
for arg in "$@"; do
  if [ "$arg" == "--check" ]; then
    CHECK_MODE=true
  fi
done

if [ "$CHECK_MODE" = true ]; then
  echo "Checking license headers across the repository..."
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
    "$REPO_ROOT"
  echo "All files have correct license headers."
else
  echo "Applying license headers across the repository..."
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
    "$REPO_ROOT"
  echo "License headers applied successfully."
fi
