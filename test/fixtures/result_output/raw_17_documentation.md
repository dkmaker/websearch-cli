---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

To set up Vite library mode for both ESM and CJS outputs with Node.js conditional exports, you need to configure your `vite.config.ts`, `tsconfig.json`, and `package.json` together.

## Vite Configuration

Configure your `vite.config.ts` to build both formats:

```typescript
import { defineConfig } from 'vite'
import { resolve } from 'path'

export default defineConfig({
  build: {
    sourcemap: true,
    lib: {
      entry: resolve(__dirname, 'src/lib/main.ts'),
      name: 'YourLibraryName',
      formats: ['cjs', 'es'],
    },
    rollupOptions: {
      external: [
        'react',
        'react-dom',
        // List all peer dependencies here
      ],
      output: {
        generatedCode: 'es2015',
      },
    },
  },
})
```

The `formats` property specifies **both CommonJS and ESM outputs**. By default, Vite builds to a `/dist` folder with filenames like `your-library.es.js` and `your-library.cjs.js`.

## TypeScript Configuration

Set up `tsconfig.json` with `moduleResolution: "bundler"`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  }
}
```

Also create `tsconfig.node.json` for the build configuration:

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

## Package.json Exports Map

Define conditional exports for Node.js to use the correct format:

```json
{
  "name": "your-library",
  "exports": {
    ".": {
      "import": "./dist/your-library.es.js",
      "require": "./dist/your-library.cjs.js",
      "types": "./dist/your-library.d.ts"
    }
  },
  "main": "./dist/your-library.cjs.js",
  "module": "./dist/your-library.es.js",
  "types": "./dist/your-library.d.ts",
  "files": ["dist"]
}
```

The `exports` field allows Node.js to select the appropriate bundle based on the module system being used. ESM consumers get the `.es.js` file, while CommonJS consumers get the `.cjs.js` file. The `types` field ensures TypeScript finds type definitions.

## Key Considerations

**Minification:** By default, Vite minifies CJS and UMD outputs but not ESM (to preserve tree-shaking). To disable minification entirely, add `build.minify: false`.

**External Dependencies:** List all peer dependencies in `rollupOptions.external` to prevent bundling them and reduce package size.

**Tree-shaking:** ESM is preferred for tree-shaking benefits, which is why it remains unminified by default.