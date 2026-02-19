---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 10
cached: false
---

Setting up a Python project with Cython extensions in a **src layout** requires careful configuration in your build system and setup files to ensure extensions are properly discovered during editable installs.

## Project Structure

Start with a standard src layout:

```
project_name/
├── src/
│   └── mypackage/
│       ├── __init__.py
│       ├── pure_module.py
│       ├── _cython_ext.pyx
│       └── _cython_ext.pxd
├── tests/
│   └── test_mypackage.py
├── pyproject.toml
├── setup.py
├── MANIFEST.in
└── README.md
```

## Configuration Files

**pyproject.toml**: Define build requirements and basic metadata:

```toml
[build-system]
requires = ["setuptools>=65.0", "wheel", "Cython>=0.29.32"]
build-backend = "setuptools.build_meta"

[project]
name = "mypackage"
version = "0.1.0"
description = "Package with Cython extensions"

[tool.setuptools.packages.find]
where = ["src"]

[tool.setuptools.package-data]
mypackage = ["*.so", "*.pyd"]
```

**setup.py**: Handle Cython extension building (required for editable installs):

```python
from setuptools import setup, Extension
from Cython.Build import cythonize
from setuptools.command.build_ext import build_ext
import os

# Ensure extensions are built in-place for editable installs
class build_ext_inplace(build_ext):
    def finalize_options(self):
        super().finalize_options()
        self.inplace = 1

extensions = [
    Extension(
        name="mypackage._cython_ext",
        sources=["src/mypackage/_cython_ext.pyx"],
        language="c",
    )
]

setup(
    ext_modules=cythonize(extensions, language_level=3),
    cmdclass={"build_ext": build_ext_inplace},
)
```

**MANIFEST.in**: Ensure Cython source files are included:

```
recursive-include src/mypackage *.pyx
recursive-include src/mypackage *.pxd
```

## Installation for Development

Install in editable mode with build isolation to trigger extension compilation:

```bash
pip install -e . --config-settings editable_mode=compat
```

The `editable_mode=compat` flag ensures extensions are built in-place within the src directory, allowing Python to discover them alongside pure modules at import time.

## Key Considerations

**Extension Discovery**: Place `.pyx` and `.pxd` files directly in your package directory (alongside `__init__.py`). After `pip install -e`, compiled extensions (`.so` on Unix, `.pyd` on Windows) will be in the same directory, making them discoverable via standard Python imports.

**Build Isolation**: Modern `pip` uses PEP 517 build isolation by default. This is beneficial here because it ensures Cython is available during the build phase. The setup.py handles the actual extension compilation.

**Import Verification**: In your code, import extensions normally:

```python
# src/mypackage/__init__.py
from mypackage._cython_ext import my_cython_function
from mypackage.pure_module import my_python_function
```

**Troubleshooting**: If extensions aren't discovered after `pip install -e`, verify:
- `.pyx` files are listed in MANIFEST.in
- The extension name in setup.py matches the module path (`mypackage._cython_ext`)
- Rebuild with `pip install -e . --force-reinstall --no-cache-dir` to force recompilation

The src layout approach prevents accidental imports from the project root and ensures your development environment matches how users will import your package after installation.