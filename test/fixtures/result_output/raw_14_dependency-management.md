---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 6
cached: false
---

The search results provided don't directly address libssl version mismatches with C extensions in Alpine Docker images. However, I can help explain this issue based on general knowledge of the problem.

## Root Cause

When a Python C extension module is **compiled against a specific version of libssl** (typically on a glibc-based Linux system like Ubuntu), but then used in an Alpine Docker image that has a different libssl version or uses musl libc instead of glibc, the import fails because the compiled binary can't find or link against the required SSL library version.

## Solutions

**1. Rebuild C extensions in Alpine**
Run the build process inside the Alpine container rather than using pre-compiled wheels:

```dockerfile
RUN apk add --no-cache python3-dev gcc musl-dev libffi-dev openssl-dev
RUN pip install --no-cache-dir --no-binary :all: your-package-name
```

The `--no-binary :all:` flag forces pip to compile from source for all packages.

**2. Use manylinux wheels instead**
Some packages provide `manylinux` wheels that work better across Linux distributions. If available, specify the wheel explicitly:

```bash
pip install your-package-name==version --only-binary :all:
```

**3. Match Alpine's libssl version to the compiled version**
Install the exact libssl version the extension was compiled against:

```dockerfile
RUN apk add --no-cache openssl==<specific-version>
```

**4. Use a glibc-compatible base image**
Instead of Alpine, use a standard Python image or a glibc-based Linux:

```dockerfile
FROM python:3.11-slim  # Instead of python:3.11-alpine
```

The simplest solution is typically option 4, as Alpine's minimal footprint often comes at the cost of compatibility with pre-compiled C extensions. If size is critical, option 1 (rebuilding in-container) is preferred despite longer build times.