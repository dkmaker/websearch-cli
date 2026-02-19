---
provider: brave
mode: web
mode_adjusted: true
truncated: false
sources_count: 5
cached: false
---

**vite-tsconfig-paths - npm**
**Give vite the ability to resolve imports using TypeScript's path mapping**. [!NOTE] New in v6 – On-demand tsconfig discovery with projectDiscovery: "lazy", automatic tsconfig reloads (including vite build --watch), support for “absolute path” ...

**Setting Up Vitest to Support TypeScript Path Aliases - Tim Santeford**
**This plugin automatically reads and maps the paths from tsconfig.json, allowing you to use them in your tests without extra configuration**. ... This plugin will save you the effort of writing manual path resolution logic and seamlessly integrates ...

**GitHub - aleclarson/vite-tsconfig-paths: Support for TypeScript's path mapping in Vite**
**Give vite the ability to resolve imports using TypeScript's path mapping**. ... New in v6 – On-demand tsconfig discovery with projectDiscovery: "lazy", automatic tsconfig reloads (including vite build --watch), support for “absolute path” ...

**Common Errors | Guide | Vitest**
Vite doesn't take into account tsconfig.json by default, so you might need to install vite-tsconfig-paths yourself, if you rely on this behaviour. ... import { defineConfig } from 'vitest/config' import tsconfigPaths from 'vite-tsconfig-paths' export default defineConfig({ plugins: [tsconfigPaths()] }) ... Make sure you don't have relative aliases.

**typescript - Vitest - @ src folder alias not resolved in test files - Stack Overflow**
import { defineConfig } from 'vitest/config'; import { svelte } from '@sveltejs/vite-plugin-svelte'; import path from 'path'; export default defineConfig({ plugins: [svelte({ hot: !process.env.VITEST })], test: { globals: true, environment: 'jsdom', }, resolve: { alias: { $lib: path.resolve(__dirname, './src/lib'), }, }, }); ... { "extends": "@vue/tsconfig/tsconfig.web.json", "include": [ "env.d.ts", "src/**/*", "src/**/*.vue" ], "exclude": [ "vitest/**/*" ], "compilerOptions": { "composite": true, "baseUrl": ".", "paths": { "@/*": [ "./src/*" ] }, "strict": true, "experimentalDecorators": true } }