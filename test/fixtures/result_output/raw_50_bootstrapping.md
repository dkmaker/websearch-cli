---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

The search results provided focus on basic TypeScript and Express setup using CommonJS and ts-node/nodemon, and do not contain information about the advanced setup you're asking for: **ESM mode, tsx, esbuild bundling, or OpenAPI schema generation from Zod validators**.

Based on my knowledge beyond these search results, here's the correct approach:

## Project Initialization

```bash
npm init -y
npm install express zod
npm install -D typescript tsx esbuild @types/express @types/node
```

## TypeScript Configuration

Create `tsconfig.json` with ESM settings:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ES2020",
    "moduleResolution": "bundler",
    "rootDir": "./src",
    "outDir": "./dist",
    "strict": true,
    "esModuleInterop": false,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": ["src"],
  "exclude": ["node_modules"]
}
```

## Development with tsx

Add to `package.json`:

```json
{
  "type": "module",
  "scripts": {
    "dev": "tsx watch src/server.ts",
    "build": "esbuild src/server.ts --bundle --platform=node --outfile=dist/server.js",
    "start": "node dist/server.js"
  }
}
```

## Production Bundling with esbuild

Create a build script or use the command above. The `--bundle` flag packages everything for production deployment.

## OpenAPI Schema Generation from Zod

For keeping OpenAPI schemas in sync with route handlers, use libraries like **zod-openapi** or **hono** with Zod integration, which automatically generate OpenAPI specs from your validators:

```typescript
import { z } from 'zod';
import { ZodOpenApiOperation } from 'zod-openapi';

const createItemSchema = z.object({
  name: z.string(),
  price: z.number(),
});

// Schema automatically enforces type safety and can generate OpenAPI docs
```

**Note:** The search results do not cover these advanced patterns. For production-grade implementation with automatic schema synchronization, consider frameworks like **Hono**, **Elysia**, or **tRPC**, which provide built-in support for Zod validation and automatic API documentation generation.