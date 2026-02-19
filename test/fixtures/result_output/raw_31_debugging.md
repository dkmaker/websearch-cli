---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

Different relevance scores for the same document across repeated searches with `random_score` occurs because **by default, `random_score` uses internal Lucene document IDs as seed values, which are not reproducible since documents can be renumbered after index merges**.

When you don't specify a seed and field, the random scoring is based on Lucene's internal document IDs, which can change whenever segments merge in the index. This is particularly problematic with uneven shard distribution, as it affects how documents are indexed and how frequently merges occur across shards.

To achieve consistent random scores across repeated searches, provide explicit `seed` and `field` parameters to the `random_score` function:

```json
{
  "query": {
    "function_score": {
      "random_score": {
        "seed": 10,
        "field": "_seq_no"
      }
    }
  }
}
```

**Recommended field choices:**

- `_seq_no`: Good for consistency but regenerates random scores when documents are updated
- A stable, unique document field that never changes: Ensures consistent scores regardless of index updates

**Additional factor affecting score changes:**

Beyond the seed issue, scores may change across searches even with consistent randomization if the index is modified. This happens because **inverse document frequency (IDF) changes when documents are added or modified**, affecting all document scores in the query results.

If you use a mutable field like `_seq_no` as your seed, the random score will regenerate whenever that document is partially updated, even if other fields change. To prevent this, use a document field that never changes.