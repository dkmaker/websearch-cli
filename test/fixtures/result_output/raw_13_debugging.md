---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 5
cached: false
---

The issue is likely **off-heap memory consumption** exceeding your pod's 2Gi limit, even though your JVM heap (-Xmx1536m) is within bounds.

Off-heap memory includes allocations made through JNI, direct buffers, and libraries like gRPC that bypass the JVM garbage collector. This memory isn't managed by the garbage collector and can grow independently of your heap size limit. Your application may use more off-heap memory under the higher load conditions typical in Kubernetes compared to your local development environment.

## Debugging Steps

**Confirm OOMKilled status:**
```
kubectl describe pod <pod-name> | grep -A 5 "Last State"
```

Look for `Reason: OOMKilled` and `Exit Code: 137`

**Check actual memory usage:**
```
kubectl top pods -n <namespace> --containers
```

Compare the actual usage to your 2Gi limit. If usage is exceeding 2Gi while heap usage remains around 1536m, off-heap memory is likely the culprit.

**Check for memory leaks:**
Monitor if memory usage trends upward over time, indicating a leak.

## Solutions

1. **Increase the pod memory limit** beyond 2Gi to account for off-heap memory. If you're using 1536m of heap plus additional off-heap, try 3Gi or higher and monitor actual usage.

2. **Reduce off-heap memory consumption** by:
   - Limiting direct buffer allocations
   - Adjusting library-specific settings (e.g., gRPC connection pool sizes)
   - Reviewing any native code or JNI usage

3. **Verify JVM runtime settings** are not conflicting with container limits. Ensure no additional JVM memory flags are being set that would increase memory usage beyond -Xmx1536m.

The discrepancy between local and Kubernetes behavior suggests your local environment has either lower concurrency, fewer connections, or less traffic triggering larger off-heap allocations in the Kubernetes deployment.