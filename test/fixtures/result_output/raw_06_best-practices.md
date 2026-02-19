---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 6
cached: false
---

For a Rust async application using tokio with multiple error sources, the recommended approach is to **define a custom error enum that encompasses all possible error types, implement the standard `Error` trait, and use `Result` types to propagate errors through tasks**.

## Core Strategy

Define a custom error type that wraps errors from different sources:

```rust
use std::error::Error;
use std::fmt;

#[non_exhaustive]
#[derive(Debug)]
pub enum AppError {
    NetworkError(String),
    DatabaseError(String),
    SerializationError(String),
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::NetworkError(msg) => write!(f, "Network error: {}", msg),
            AppError::DatabaseError(msg) => write!(f, "Database error: {}", msg),
            AppError::SerializationError(msg) => write!(f, "Serialization error: {}", msg),
        }
    }
}

impl Error for AppError {}
```

The **`#[non_exhaustive]` attribute** enables forward compatibility by preventing users from exhaustively matching all variants if you add new error types later.

## Async Task Error Propagation

For tasks spawned with `tokio::spawn`, always return `Result` types rather than trying to return custom errors directly. Since spawned tasks must return a type that doesn't prevent the task from being awaited, use a wrapper:

```rust
let task = tokio::spawn(async move {
    perform_operation().await
        .map_err(|e| AppError::NetworkError(e.to_string()))
});

let result = task.await;
```

## Integration with Multiple Sources

Implement `From` traits to convert errors from different libraries into your custom type:

```rust
impl From<std::io::Error> for AppError {
    fn from(err: std::io::Error) -> Self {
        AppError::NetworkError(err.to_string())
    }
}

impl From<serde_json::Error> for AppError {
    fn from(err: serde_json::Error) -> Self {
        AppError::SerializationError(err.to_string())
    }
}
```

This allows you to use the `?` operator seamlessly, which automatically propagates and converts errors.

## Using anyhow for Simpler Cases

For applications where you need flexible error handling without strict type checking, consider using the `anyhow` crate with context methods:

```rust
use anyhow::{Result, Context};

async fn process_data() -> Result<Data> {
    let data = fetch_from_network()
        .await
        .context("Failed to fetch data from network")?;
    
    let parsed = serde_json::from_str(&data)
        .context("Failed to deserialize response")?;
    
    Ok(parsed)
}
```

## Best Practices

- **Libraries should never panic**; instead, propagate errors to callers
- Allow downstream users to extract the underlying errors if needed
- Use `Result` types with `?` operator for ergonomic error propagation in async functions
- For concurrent tasks with multiple potential failures, use patterns like `tokio::try_join!` to aggregate results efficiently