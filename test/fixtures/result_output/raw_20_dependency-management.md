---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

Cargo's **resolver attempts to unify common dependencies to specific versions**, so two crates depending on different versions of serde will typically trigger a conflict rather than allow both versions to coexist in a single workspace. When the resolver encounters incompatible version requirements, it will backtrack and error if no solution is found.

## Primary Solutions

**Use the `[patch]` table** to enforce a single serde version across your workspace. Add this to your workspace root `Cargo.toml`:

```toml
[patch.crates-io]
serde = { version = "1.0.X" }  # Specify the compatible version
```

This forces all dependencies to use the same serde version, which solves the derive macro incompatibility issue by ensuring consistent macro expansion across your crates.

**Align dependency versions** by reviewing which crates require which serde versions and updating their `Cargo.toml` files to use compatible versions when possible. The goal is to make both crates depend on the same serde version that satisfies both requirements.

**Use `cargo tree`** to visualize and diagnose the dependency conflict. Run:

```bash
cargo tree --workspace --all-features --invert serde
```

This shows exactly which packages are pulling in different serde versions, helping you identify where to apply patches.

## Workspace Structure Consideration

If you have **binary (`bin`) or dynamic library (`dylib`) crates** in your workspace that genuinely cannot share the same serde version, you may need to extract conflicting crates into separate workspaces or build them independently, though this complicates your workflow.

The derive macro incompatibility specifically occurs because different serde versions generate different code at compile time—unifying to a single version is the standard approach to resolve this.