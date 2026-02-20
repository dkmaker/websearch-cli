# Release Workflow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** GitHub Actions workflow that builds and releases cross-platform binaries when a version tag is pushed.

**Architecture:** GoReleaser handles cross-compilation and GitHub Release creation. A minimal GitHub Actions workflow triggers on `v*` tag push and runs GoReleaser.

**Tech Stack:** GitHub Actions, GoReleaser, Go 1.24

---

### Task 1: Create GoReleaser Configuration

**Files:**
- Create: `.goreleaser.yaml`

**Step 1: Create `.goreleaser.yaml`**

```yaml
version: 2

builds:
  - main: ./cmd/websearch/
    binary: websearch
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
```

**Step 2: Validate config locally**

Run: `go install github.com/goreleaser/goreleaser/v2@latest && goreleaser check`
Expected: config is valid

**Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "chore: add goreleaser configuration"
```

---

### Task 2: Create GitHub Actions Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Create `.github/workflows/release.yml`**

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add release workflow with goreleaser"
```

---

### Task 3: Verify with Dry Run

**Step 1: Run GoReleaser dry run**

Run: `goreleaser release --snapshot --clean`
Expected: Builds 6 binaries (linux/darwin/windows x amd64/arm64), creates archives in `dist/`

**Step 2: Verify archives exist**

Run: `ls dist/*.tar.gz dist/*.zip`
Expected: 4 `.tar.gz` files (linux + darwin) and 2 `.zip` files (windows)

**Step 3: Clean up**

Run: `rm -rf dist/`

---
