---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 6
cached: false
---

The continuous MOVED and ASK redirections during cluster rebalancing are expected behavior, but the latency spikes indicate your go-redis client may not be handling the redirections efficiently or the cluster topology is not updating properly.

## Why This Happens

During cluster rebalancing after adding a new node, Redis permanently reassigns hash slots to the new node, causing redirections. The **MOVED** response indicates a permanent slot assignment change, while **ASK** indicates temporary redirection during the actual migration. These are not errors but redirection responses telling the client that keys have moved to different nodes.

## Why You're Experiencing Latency Spikes

Your latency spikes likely stem from one of these issues:

**Stale slot cache**: If go-redis's slot-to-node mapping becomes outdated, it continues sending requests to the wrong node, forcing multiple round-trips before reaching the correct node. Each MOVED response should trigger a slot map update, but if this isn't happening properly, you'll see repeated redirections for the same keys.

**Topology refresh delays**: If the client isn't refreshing its cluster topology frequently during resharding, it accumulates stale routing information. The cluster specification notes that clients should update local slot mappings when receiving MOVED redirections to avoid repeated errors.

**Normal resharding overhead**: During slot migration, some requests may fail temporarily as keys transition between nodes, particularly after the receiving node executes RESTORE-ASKING commands. This is inherent to the resharding process.

## Solutions for Go-Redis

Ensure go-redis is configured properly for cluster mode and enable automatic topology refresh during rebalancing:

- Verify you're using `redis.NewClusterClient()` rather than a single-node client, as cluster-aware clients automatically handle MOVED and ASK redirections.
- Configure topology refresh intervals aggressively during the rebalancing window to detect slot assignment changes quickly.
- Check that your cluster client configuration includes proper retry logic and cluster retry strategies to handle temporary redirections smoothly.

The redirections should diminish once resharding completes and all nodes finish migrating their slots.