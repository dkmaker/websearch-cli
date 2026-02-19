---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

The provided search results do not contain specific guidance on bootstrapping a Python monorepo using **uv or poetry workspaces** with path dependencies. However, I can address what the results do cover about Python monorepo setup.

## What the Search Results Address

The search results emphasize using **virtual environments for dependency isolation** in Python monorepos. Specifically, the `venv` module from Python's standard library can create isolated environments for each project to manage project-specific dependencies separately, preventing conflicts between requirements.

For monorepo management at scale, the results recommend build tools like **Pants** (which supports Python and offers caching and dependency optimization) or **Earthly** (useful for containerized services). These tools help orchestrate builds across multiple packages and can optimize performance by caching and skipping unchanged parts of the codebase.

The results also reference general monorepo structure principles: defining a clear directory structure with top-level directories for each project and shared resources.

## Gaps in Available Information

The search results do not cover:
- Poetry workspace configuration with path dependencies
- uv's monorepo handling capabilities
- Specific bootstrapping procedures for either tool
- How to configure interdependent packages in these tools

For accurate guidance on uv or poetry workspaces with path dependencies, you would need documentation from those tools' official resources or community guides specifically focused on workspace configuration and local package dependencies.