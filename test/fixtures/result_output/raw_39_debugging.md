---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 7
cached: false
---

The search results provided do not contain specific information about **SnapStart** or Spring Boot 3 initialization patterns with provisioned concurrency. However, the results do explain general reasons why cold start latency can still occur even with provisioned concurrency enabled.

When cold starts occur at exactly the provisioned concurrency limit, the most likely cause is that **your traffic is exceeding the provisioned concurrency level**. Once Lambda must scale beyond your provisioned instances, any additional concurrent invocations will trigger cold starts. You should verify that your provisioned concurrency value accounts for your actual peak concurrent requests rather than just peak traffic volume.

Additionally, several other factors can cause cold starts despite provisioned concurrency:

- **Provisioned concurrency applied to wrong version**: Provisioned concurrency must be applied to a specific function version or alias, not the $LATEST version. If you're invoking the wrong version, you'll experience cold starts.

- **Initialization code outside the handler**: Code executed outside the main Lambda handler—including database connection pool initialization—runs on every cold start and is often responsible for the largest share of cold start latency. Even with provisioned concurrency, any new environments created during scaling will execute this code.

- **Infrastructure refreshes**: Lambda's internal infrastructure occasionally refreshes containers, which can cause cold starts even within the provisioned pool.

To address your Spring Boot initialization specifically, the search results indicate that initializing database connections during the init phase significantly contributes to cold start latency. You may need to defer non-critical initialization or optimize your connection pool setup, though the search results don't contain SnapStart-specific guidance for addressing this issue.