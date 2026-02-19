---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 7
cached: false
---

Migrating from Yarn 1 workspaces to pnpm while handling implicit hoisted dependency resolution requires identifying phantom dependencies and explicitly declaring them before migration.

## The Core Challenge

In Yarn 1, packages can access dependencies that are hoisted to the root `node_modules` even if they're not declared in their individual `package.json` files. pnpm enforces **strict dependency resolution**, allowing packages access only to dependencies explicitly declared in their `package.json`. This is the primary breaking point during migration.

## Migration Steps

**1. Audit for phantom dependencies**

Before migrating, scan your packages for undeclared dependency usage. This is critical—without doing this first, your build will fail after switching to pnpm.

**2. Install and set up pnpm**

Use Corepack (included with modern Node.js) or the standalone installation script to avoid version conflicts.

**3. Create the workspace configuration**

Create a `pnpm-workspace.yaml` file in your repository root using glob patterns to define package locations.

**4. Convert your lock file**

Run `pnpm import` to convert your existing `yarn.lock` to `pnpm-lock.yaml`, preserving dependency resolution from Yarn.

**5. Handle hoisted dependencies**

For packages with implicit dependencies, you have two approaches:

- **Declare missing dependencies**: Add undeclared dependencies to each package's `package.json` that relies on them. This is the recommended approach for long-term maintainability.
- **Use pnpm overrides**: As a temporary workaround, use pnpm's override feature in the root `package.json` to force specific versions, though this requires ongoing lockfile management.

**6. Use the workspace protocol for internal packages**

Replace internal package references with `"internal-pkg": "workspace:*"` to ensure packages resolve from your workspace rather than the public registry, preventing version drift.

**7. Remove old lock files**

Delete `yarn.lock` and `package-lock.json` files after verifying pnpm resolution works correctly.

## Lessons from Large-Scale Migration

The CKEditor team's migration to pnpm from Yarn Classic showed that while installation speed improved dramatically (80 seconds to 16 seconds for cold installs on CI), handling complex dependency trees required careful attention to dependency declarations and occasional use of overrides. They noted that explicitly managing dependencies is "a small trade-off for the benefits" of stricter dependency control and consistent environments.