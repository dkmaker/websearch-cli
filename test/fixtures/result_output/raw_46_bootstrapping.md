---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

To set up a TypeScript monorepo with project references where shared packages serve both Next.js and NestJS applications while maintaining incremental compilation, you'll need a structured configuration across multiple `tsconfig.json` files and proper workspace setup.

## Root Configuration

Your root `tsconfig.json` should orchestrate all sub-projects without compiling directly:

```json
{
  "files": [],
  "references": [
    { "path": "./packages/shared" },
    { "path": "./apps/api" },
    { "path": "./apps/web" }
  ]
}
```

The empty `files` array is critical—the root config only orchestrates builds, it doesn't compile anything.

## Shared Package Configuration

Your shared package must enable **`composite: true`** along with declaration generation to ensure type information is available to dependent projects without requiring source file access:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "composite": true,
    "declaration": true,
    "declarationMap": true,
    "incremental": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "skipLibCheck": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

The `composite`, `declaration`, and `declarationMap` options are essential. TypeScript creates `.d.ts` declaration files and `.tsbuildinfo` files for incremental tracking. The `declarationMap` ensures editor features like "Go to Definition" work correctly.

## Application Configurations

Both your NestJS and Next.js apps should reference the shared package:

```json
{
  "extends": "../../tsconfig.json",
  "compilerOptions": {
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "paths": {
      "@shared/*": ["../../packages/shared/src/*"]
    }
  },
  "include": ["src/**/*"],
  "references": [
    { "path": "../../packages/shared" }
  ]
}
```

Use `extends` to inherit root compiler options and avoid duplication. Path aliases on a per-project basis (rather than globally) prevent conflicts in a monorepo.

## Package Manager Workspaces

Configure your `package.json` to define workspace boundaries:

```json
{
  "name": "my-monorepo",
  "private": true,
  "workspaces": [
    "packages/*",
    "apps/*"
  ],
  "scripts": {
    "build": "tsc --build",
    "build:watch": "tsc --build --watch",
    "clean": "tsc --build --clean",
    "type-check": "tsc --noEmit"
  }
}
```

Use `tsc --build` to respect project dependencies and enable incremental compilation.

## Importing Across Projects

Import from the shared package using your configured path alias:

```typescript
// apps/api/src/index.ts
import { formatDate, parseConfig } from "@shared/utils";

// Or use relative paths if aliases aren't configured
import { formatDate } from "../../packages/shared/src/utils";
```

TypeScript resolves these imports through the referenced project's declaration files, not source files directly.

## Preventing Type Declaration Duplication

TypeScript automatically generates `.d.ts` files in your shared package's `dist` directory during the build. Both applications consume these same declarations, eliminating duplication. The `incremental` flag tracks changes via `.tsbuildinfo` files, so only modified projects rebuild.

## Performance Optimization

For larger monorepos where your editor (VSCode) may experience slowdowns, enable `disableSourceOfProjectReferenceRedirect` in your project's `compilerOptions`. However, ensure TypeScript files are recompiled when changed by running the compiler in watch mode.

## Build Tools Integration

If using Nx or Turborepo alongside project references, configure your build tool to respect TypeScript dependencies:

```json
{
  "pipeline": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**"]
    }
  }
}
```

This ensures builds respect the project reference graph.

## When Project References Are Worth It

Project references are most beneficial for large monorepos (100+ projects) or repositories with many contributors, as they significantly reduce CI type-checking times. For smaller setups, they may introduce unnecessary complexity.