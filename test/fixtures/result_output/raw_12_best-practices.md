---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

**Graceful shutdown in a Go HTTP server with background Kafka consumers requires coordinating signal handling, HTTP server shutdown, and goroutine termination using contexts and synchronization primitives.**

The fundamental approach involves three coordinated steps: capturing termination signals, signaling all components to stop accepting new work, and waiting for in-flight operations to complete within a timeout.

## Signal Handling and Coordination

Start by capturing SIGTERM and SIGINT signals, then create a shared context that signals all components—both the HTTP server and background workers—simultaneously. Use `signal.NotifyContext` to create a cancellable context that automatically cancels when these signals arrive:

```go
mainCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

Alternatively, use the lower-level `signal.Notify` approach with explicit context cancellation.

## HTTP Server Shutdown

Stop the HTTP server from accepting new requests by calling `server.Shutdown(ctx)` with a timeout context. This method closes all open listeners and waits indefinitely for active connections to return to idle. Set a reasonable shutdown timeout between 30-60 seconds:

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := server.Shutdown(shutdownCtx); err != nil {
    log.Println("HTTP shutdown error:", err)
}
```

**Important:** In containerized environments, add a brief delay before initiating shutdown to allow load balancers to stop sending traffic.

## Background Kafka Consumer Shutdown

For background workers consuming from Kafka with manual offset commits, signal them through the shared context and wait for graceful completion. When the context is cancelled, your consumer goroutines should:

1. Stop consuming new messages
2. Finish processing in-flight messages
3. Commit final offsets before exiting

Use a `WaitGroup` to synchronize goroutine completion:

```go
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    for {
        select {
        case <-mainCtx.Done():
            // Stop consuming and commit pending offsets
            return
        default:
            // Consume and process messages
        }
    }
}()

// After signals are received:
wg.Wait() // Wait for all workers to finish
```

## Complete Pattern

Combine components using `errgroup` for clean concurrent error handling:

```go
g, gCtx := errgroup.WithContext(mainCtx)

g.Go(func() error {
    return httpServer.ListenAndServe()
})

g.Go(func() error {
    <-gCtx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return httpServer.Shutdown(shutdownCtx)
})

// Wait for all goroutines
if err := g.Wait(); err != nil && err != http.ErrServerClosed {
    log.Fatal(err)
}
```

## Best Practices

**Stop accepting new work first** before waiting for in-flight operations. **Close resources in reverse order of creation**—in this case, stop consuming from Kafka, allow in-flight messages to finish processing and be committed, then close the HTTP server. Always set a timeout to prevent indefinite hangs, and **log shutdown progress for debugging**.