---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

The performance degradation occurs because **ANALYZE updated table statistics, causing the planner to switch from a sequential scan (efficient for large tables) to a nested loop join with index scans (inefficient when the outer relation is large)**.

## Why This Happened

After ANALYZE, PostgreSQL gathered new statistics that made the planner estimate the outer relation as smaller or more selective than before. Since **nested loop joins are particularly efficient when the outer relation is small**, the optimizer chose this strategy. However, if your outer relation is actually large, nested loop joins become very inefficient because the inner index scan must execute repeatedly—once for every outer row.

## Why Performance Degraded

The sequential scan was faster because it performs **single I/O operations that retrieve multiple rows in a block**. In contrast, **index scans require multiple I/O operations per row**: looking up the row in the index, then fetching the actual row from the heap. On a large partitioned table, this random I/O access pattern is significantly slower than the sequential I/O of a table scan.

## Potential Solutions

1. **Force a different join strategy** temporarily to identify the optimal plan:
   - Disable nested loop joins: `SET enable_nestloop = off;` to try hash or merge joins
   - Disable index scans on the inner relation if they're being used inefficiently

2. **Update statistics more intelligently**: Run `ANALYZE` on specific columns involved in the join with higher sampling if default statistics are misleading

3. **Check for stale or incorrect statistics**: Verify that the table statistics reflect the actual data distribution, especially for partitioned tables where per-partition statistics matter

4. **Consider index optimization**: If you must use an index scan, ensure it's selective enough to justify its random I/O cost relative to sequential scanning

The core issue is that ANALYZE changed the optimizer's cost estimates; adjusting statistics accuracy or explicitly guiding the planner can restore performance.