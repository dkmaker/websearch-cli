---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

Configuring Istio for this migration scenario requires a **phased approach using PeerAuthentication modes** rather than jumping directly to STRICT mode.

## Initial Setup: cert-manager Integration

First, integrate cert-manager with Istio by deploying **istio-csr**, which handles certificate signing requests for all mesh workloads. Configure a cert-manager Issuer in the `istio-system` namespace and update your Istio installation to use the cert-manager CA server:

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
spec:
  values:
    global:
      caAddress: cert-manager-istio-csr.cert-manager.svc:443
  components:
    pilot:
      k8s:
        env:
        - name: ENABLE_CA_SERVER
          value: "false"
```

This configuration disables istiod's built-in CA, routing all certificate management through cert-manager for automatic rotation.

## Migration Strategy: PERMISSIVE → STRICT

By default, Istio configures workloads in **PERMISSIVE mode**, which allows services to accept both plain text and mTLS connections simultaneously. This is your migration bridge: legacy non-mesh services can continue sending unencrypted traffic while mesh services communicate with mTLS.

Once legacy services are migrated to the mesh or decommissioned, transition to STRICT mode by applying a PeerAuthentication policy:

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: "default"
spec:
  mtls:
    mode: STRICT
```

## Gateway Approach for Mixed Traffic

For applications that require both open ingress from external services and strict internal mTLS, use an **Istio Ingress Gateway** as your entry point. The gateway can accept unencrypted traffic from legacy services while enforcing mTLS on all internal service-to-service communication.

## Key Considerations

Cert-manager integration automatically handles certificate renewal for workload certificates through istio-csr, eliminating manual certificate management during your migration period. Note that some documentation suggests this approach adds complexity, with native Istio CA rotation being recommended for mature deployments, but cert-manager provides flexibility for heterogeneous environments during migration.