---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 10
cached: false
---

For a dual-package TypeScript library supporting both ESM and CJS with Node.js native ESM loader compatibility, you need **two separate tsconfig files** rather than a single configuration.

## ESM Configuration (tsconfig.json)

```json
{
  "compilerOptions": {
    "module": "ESNext",
    "moduleResolution": "NodeNext",
    "target": "ES2018",
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./module/",
    "strict": true,
    "esModuleInterop": true,
    "newLine": "LF"
  },
  "include": ["src/**/*"]
}
```

## CommonJS Configuration (tsconfig.cjs.json)

```json
{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "module": "CommonJS",
    "moduleResolution": "Node",
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./cjs/"
  }
}
```

## Key Configuration Points

- **moduleResolution**: Use `"NodeNext"` for ESM to enable Node.js native ESM loader support with proper specifier resolution
- **module**: Set to `"ESNext"` for ESM output to preserve `.js` extensions in import paths, and `"CommonJS"` for CJS
- **declaration/declarationMap**: Include both to generate `.d.ts` files with source maps for type definitions
- **sourceMap**: Enables source map generation for debugging
- **outDir**: Separate output directories (`module/` for ESM, `cjs/` for CJS) allow you to publish both formats in a single package using subdirectory package.json files with appropriate `"type"` fields

After compilation with both configs, run `tsconfig-to-dual-package` to generate package.json files in each output directory. Your build script should be:

```json
{
  "scripts": {
    "build": "tsc -p ./tsconfig.json && tsc -p ./tsconfig.cjs.json && tsconfig-to-dual-package"
  }
}
```

This setup ensures Node.js correctly interprets ESM files in the `module/` directory and CommonJS files in the `cjs/` directory based on their respective package.json `"type"` declarations.