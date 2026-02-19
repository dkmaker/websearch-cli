---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 7
cached: false
---

The **"Type instantiation is excessively deep and possibly infinite"** error occurs when TypeScript's type resolver encounters recursive or deeply nested types that exceed the compiler's instantiation depth limit. In your scenario with a recursive conditional type implementing deep partial on complex nested API responses, the issue arises because the mapped type recursively transforms nested objects without a proper termination condition, causing the type checker to loop indefinitely.

## Root Causes

**Unbounded recursion without base cases:** When a recursive conditional type lacks a clear stopping point, the compiler cannot determine when to stop resolving the type. For example, a `DeepReadonly` mapped type that unconditionally recurses on nested objects will trigger this error as it processes deeply nested API response types.

**Complex nested generics:** Types with heavily nested generic parameters can overload TypeScript's type system, particularly when combined with recursive conditional logic.

## Solutions

**Add depth limiting:** Introduce a counter type parameter that stops recursion at a specific depth. Instead of unbounded recursion, use a depth counter that decrements with each recursion level:

```typescript
// GOOD: Limit recursion depth using a counter tuple
type Depth = [never, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

type DeepPartialLimited<T, D extends Depth = Depth> = D extends 0
  ? T // Base case: stop at depth 0
  : T extends object
  ? { [P in keyof T]?: DeepPartialLimited<T[P], Depth[Exclude<D, D>]> }
  : T;
```

**Use type aliases and split complex types:** Break down intricate recursive types into simpler components using intermediate type aliases, which enables type caching and reduces resolution complexity.

**Simplify conditional types:** Keep conditional type expressions as shallow as possible to avoid confusing the type system. Complex nested conditionals compound the recursion problem.

## Debugging Strategies

**Isolate the problem:** Comment out sections of your type definition to find which nested level or property causes the issue.

**Add type checkpoints:** Insert intermediate type aliases to identify where type resolution fails.

**Use TypeScript Playground:** Hover over intermediate types in the Playground to inspect resolved values and find the exact point where the type becomes excessively deep.

For library use cases like Kysely, you can use assertion methods like `$assertType` to help TypeScript replace complex nested helper types with simpler asserted types, which doesn't reduce type safety.