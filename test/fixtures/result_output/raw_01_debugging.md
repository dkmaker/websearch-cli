---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 5
cached: false
---

Your program deadlocks because **an unbuffered channel requires an active receiver before the sender can proceed, and if no receiver exists, the sender goroutine blocks indefinitely**. When combined with `sync.WaitGroup`, the main goroutine may be waiting for all child goroutines to complete via `wg.Wait()`, but those child goroutines are stuck trying to send to the channel, creating a circular wait scenario.

Here's what happens:

1. **The sender blocks**: When a goroutine attempts to send to an unbuffered channel (`ch <- value`) and no goroutine is actively receiving from that channel, the sender blocks.

2. **WaitGroup never completes**: If the sending goroutine is blocked on the channel operation, it never reaches `wg.Done()` (or `defer wg.Done()` never executes). The `WaitGroup` counter never decrements to zero.

3. **Main goroutine deadlocks**: Your main thread calling `wg.Wait()` blocks indefinitely waiting for a counter that will never reach zero.

To fix this, you have two options:

**Option 1: Add a concurrent receiver**
Create a separate goroutine that reads from the channel while your sender writes to it. This allows the sender to complete without blocking.

**Option 2: Use a buffered channel**
Change `make(chan int)` to `make(chan int, n)` where `n` is the buffer size. This allows the sender to write without an immediate receiver, up to the buffer capacity.

**Option 3: Remove the channel entirely**
If you only need synchronization (not data exchange), use `sync.WaitGroup` alone without channels, as channels add unnecessary complexity and deadlock risk.

The critical lesson: **always ensure there's a corresponding consumer for every channel sender, or use a buffered channel to decouple the send from the receive**.