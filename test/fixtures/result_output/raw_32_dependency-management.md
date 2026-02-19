---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 9
cached: false
---

To set up a Nix flake for a polyglot project with Python, Rust, and Node.js, define your dependencies in a shared `devShells.default` output using `buildInputs` to include both language runtimes and system libraries.

## Basic Structure

Create a `flake.nix` file with inputs for each language toolchain and the system libraries your Rust crate needs:

```nix
{
  description = "Polyglot dev environment with Rust, Python, Node.js";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    rust-overlay.url = "github:oxalica/rust-overlay";
  };

  outputs = { self, nixpkgs, flake-utils, rust-overlay }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        overlays = [ (import rust-overlay) ];
        pkgs = import nixpkgs { inherit system overlays; };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Rust
            rust-bin.stable.latest.default
            
            # Python
            python3
            
            # Node.js
            nodejs
            
            # System libraries for Rust linking
            pkg-config
            openssl
            # Add other C libraries your Rust crate needs
          ];
        };
      }
    );
}
```

## Key Principles for Rust System Library Linking

**Declare all C library dependencies in `buildInputs`**: Include both the library itself (e.g., `openssl`) and build tools like `pkg-config` that Rust build scripts use to locate them. The flake in the FiveOneFour example includes libraries like `rdkafka` and `openssl` directly in `buildInputs`, which makes them available to the Rust compiler.

**Use composability through inputs**: Rather than defining Rust tooling manually, import community-maintained solutions like `rust-overlay`. This ensures all developers get consistent versions of the Rust compiler and associated tools.

**Environment reproducibility**: The `flake.lock` file locks all input versions, guaranteeing that every developer receives identical versions of dependencies—including system libraries—when they run `nix develop`.

## Activation

Developers enter the environment with:

```bash
nix develop
```

Or automatically with nix-direnv by adding `use flake` to a `.envrc` file in the project root. This approach eliminates environment drift that occurs when developers manually install different versions of tools and libraries.

The search results indicate this pattern works well for polyglot monorepos, though managing very large polyglot projects may introduce additional complexity.