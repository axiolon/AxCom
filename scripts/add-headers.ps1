# Copyright 2026 Axiolon Labs
# SPDX-License-Identifier: Apache-2.0

$ErrorActionPreference = "Stop"

# Get repository root
$repoRoot = git rev-parse --show-toplevel 2>$null
if (-not $repoRoot) {
    $repoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
}

# Check if checking or adding
$checkMode = $false
foreach ($arg in $args) {
    if ($arg -eq "--check") {
        $checkMode = $true
    }
}

if ($checkMode) {
    Write-Host "Checking license headers across the repository..."
    go run github.com/google/addlicense@v1.1.1 `
      -check `
      -c "Axiolon Labs" `
      -l apache `
      -s=only `
      -ignore "**/node_modules/**" `
      -ignore "**/dist/**" `
      -ignore "**/.next/**" `
      -ignore "**/.git/**" `
      -ignore "**/vendor/**" `
      -ignore "**/.idea/**" `
      -ignore "**/.vscode/**" `
      -ignore "**/*.json" `
      -ignore "**/*.md" `
      -ignore "**/LICENSE" `
      -ignore "**/NOTICE" `
      -ignore "**/Dockerfile" `
      $repoRoot
    Write-Host "All files have correct license headers."
} else {
    Write-Host "Applying license headers across the repository..."
    go run github.com/google/addlicense@v1.1.1 `
      -c "Axiolon Labs" `
      -l apache `
      -s=only `
      -ignore "**/node_modules/**" `
      -ignore "**/dist/**" `
      -ignore "**/.next/**" `
      -ignore "**/.git/**" `
      -ignore "**/vendor/**" `
      -ignore "**/.idea/**" `
      -ignore "**/.vscode/**" `
      -ignore "**/*.json" `
      -ignore "**/*.md" `
      -ignore "**/LICENSE" `
      -ignore "**/NOTICE" `
      -ignore "**/Dockerfile" `
      $repoRoot
    Write-Host "License headers applied successfully."
}
