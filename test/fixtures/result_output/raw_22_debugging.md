---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

When **external DNS resolution fails with SERVFAIL** while **internal service discovery works**, the issue typically stems from **DNS forwarding misconfiguration or upstream DNS server connectivity problems**, not CoreDNS itself.

## Why Internal Works But External Fails

Internal DNS resolution works because CoreDNS uses its built-in **kubernetes plugin** to query the Kubernetes API directly for cluster services. External domain resolution, however, requires CoreDNS to forward queries to upstream DNS servers configured in the `/etc/resolv.conf` file or CoreDNS configmap. When these upstream servers are unreachable, misconfigured, or decommissioned, you get SERVFAIL errors specifically for external queries.

## Primary Causes for Intermittent Failures

**Upstream DNS server issues**: The DNS forward servers configured in CoreDNS fail to respond intermittently due to:
- Upstream DNS servers being overloaded or experiencing latency
- Transient network connectivity problems to external DNS servers
- Misconfigured forwarding addresses in the CoreDNS configmap pointing to invalid or decommissioned nameservers

**Kubelet resolv.conf configuration**: If kubelet's `--resolv-conf` flag is not set correctly, pods may use an incorrect or stale `/etc/resolv.conf` file from the host, causing intermittent resolution failures when those DNS servers become unavailable.

## Troubleshooting Steps

**Test external DNS explicitly**:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup google.com
```
If this fails with SERVFAIL while internal resolution (`nslookup kubernetes.default`) succeeds, confirm the upstream DNS issue.

**Verify CoreDNS forwarding configuration**:
```bash
kubectl get configmap coredns -n kube-system -o yaml | grep forward
```
Check that the configured forwarding addresses are valid and reachable.

**Check pod-level DNS policies**:
```bash
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 10 "dnsPolicy"
```
If `dnsPolicy` is set to `Default` or `None`, the pod may be using the node's DNS resolver instead of CoreDNS.

**Optimize ndots configuration**: High `ndots` values (default is 5) cause excessive DNS queries for external domains. Reducing this can improve reliability:
```yaml
dnsPolicy: ClusterFirst
dnsConfig:
  options:
  - name: ndots
    value: "1"
```
Use fully qualified domain names (with trailing dot) for external queries to bypass search path expansion.

**Ensure kubelet configuration**: On all nodes using local DNS resolvers, verify the kubelet configuration includes the proper `--resolv-conf` flag pointing to your host's valid DNS resolver.