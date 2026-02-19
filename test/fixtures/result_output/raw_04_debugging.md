---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 5
cached: false
---

The search results don't directly address SIGABRT crashes when upgrading from Node.js v18 to v22 with native addons, but they do provide relevant context about signal handling and native addon compilation issues.

**Relevant information from the results:**

SIGABRT crashes with native addons have been documented historically. The **segfault-raub package** is specifically designed to report debug information on process crashes including SIGABRT events, suggesting this is a known challenge with native modules. The **close-with-grace package** shows that SIGABRT can be handled via signal handlers, but notes that unhandled skipped events may cause unexpected crashes or exits.

A potential factor in your upgrade is how **node-gyp detects and downloads Node.js headers**. When upgrading major versions (v18 to v22), node-gyp downloads headers or source code specific to your target version, which could introduce compatibility issues if your native addon's C++ code relies on APIs or internal structures that changed between these versions.

Historical context from the search results shows that similar projects (iobroker.smartmeter and smartmeter-obis) experienced SIGABRT/SIGSEGV crashes with native dependencies and required updating to newer Node.js versions with each major release, suggesting API stability concerns between major versions.

**Gaps in the search results:**

The results don't explain specific breaking changes between v18 and v22, strategies for diagnosing the missing stack trace, or Node.js version compatibility patterns for native addons. You would need to consult Node.js release notes or your addon's compatibility documentation for the specific cause.