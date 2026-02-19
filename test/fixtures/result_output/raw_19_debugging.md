---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 6
cached: false
---

The **Invalid hook call error occurs because the remote component and host provider are using different instances of React**, preventing the context from being properly shared across module boundaries. Even with singleton configuration, the eager loading or version mismatch causes the context object to be null when the remote component tries to consume it.

## Root Cause

In Module Federation, when a remote container exposes a component that uses `useContext`, that component needs access to the same React instance and context provider as the host application. Without proper singleton and eager configuration, each application loads its own React instance, breaking the context chain. This manifests as either:

- **"Cannot read properties of null (reading 'useContext')"** – the context is undefined
- **"Invalid hook call"** – React instances are mismatched

## Solution

Configure your **Module Federation plugin with singleton and eager flags** for both React and React-DOM in all applications (host and remote):

```javascript
new ModuleFederationPlugin({
  name: 'remoteApp',
  filename: 'remoteEntry.js',
  exposes: {
    './RemoteComponent': './src/RemoteComponent'
  },
  shared: {
    react: {
      singleton: true,
      eager: true,
      requiredVersion: '^18.2.0'
    },
    'react-dom': {
      singleton: true,
      eager: true,
      requiredVersion: '^18.2.0'
    }
  }
})
```

The key parameters are:

- **`singleton: true`** – ensures only one instance of React is loaded across all federated modules
- **`eager: true`** – loads the shared dependency immediately instead of asynchronously, preventing timing issues where the provider hasn't initialized
- **`requiredVersion`** – enforces version compatibility

## Critical Step

Apply this same configuration to **both your host and remote webpack configurations**. The host must also specify React as shared with singleton and eager flags to ensure it loads React first. Inconsistent configurations between host and remote modules will cause the context provider to be unavailable to remote components.