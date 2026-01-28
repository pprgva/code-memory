---
title: Hybrid Search
description: Combine vector and text search with RRF
---

Hybrid search combines vector similarity with text matching using Reciprocal Rank Fusion (RRF). This improves results for queries containing exact identifiers or keywords.

## How It Works

```
Query: "handleAuth authentication"
        │
        ├─► Vector Search ──► [chunk_a, chunk_b, chunk_c]
        │                      rank 0    rank 1    rank 2
        │
        └─► Text Search ────► [chunk_b, chunk_d, chunk_a]
                               rank 0    rank 1    rank 2
        │
        ▼
    RRF Fusion: score = Σ 1/(k + rank)
        │
        ▼
    [chunk_b, chunk_a, chunk_c, chunk_d]
     (appears in both lists = higher score)
```

1. **Vector search**: Semantic similarity via embeddings (existing behavior)
2. **Text search**: Simple keyword matching in chunk content
3. **RRF fusion**: Combines rankings from both sources

## Configuration

Disabled by default. Enable in `.grepai/config.yaml`:

```yaml
search:
  hybrid:
    enabled: true
    k: 60   # RRF constant (default: 60)
```

### The `k` parameter

The RRF formula is: `score(doc) = Σ 1/(k + rank_i)`

- **Higher k** (e.g., 100): More weight to documents appearing in multiple lists
- **Lower k** (e.g., 30): More weight to top-ranked documents in each list
- **Default (60)**: Balanced weighting

## When to Enable

Enable hybrid search when:

- Queries often include exact function/class names
- You mix natural language with identifiers (e.g., "handleAuth function")
- Vector-only search misses obvious keyword matches
- You search for specific variable or method names

## Examples

| Query | Vector-only | Hybrid |
|-------|-------------|--------|
| `handleUserLogin` | May miss if embedding doesn't capture identifier | Finds exact matches |
| `authentication flow` | Good semantic match | Same quality |
| `validateEmail function in user module` | Partial match | Better: combines semantic + keyword |

## Performance Note

Hybrid search loads all chunks into memory for text matching. For very large indexes (100k+ chunks), this may add latency.

Consider keeping it disabled for:
- Very large monorepos
- Purely semantic queries (no identifiers)
- Performance-critical use cases

## Technical Details

### Text Search

Simple keyword matching:
- Query is tokenized into words (lowercase, min 2 chars)
- Each chunk is scored by: `matches / total_words`
- Results sorted by score

### RRF Fusion

[Reciprocal Rank Fusion](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf) merges ranked lists:

```
score(doc) = Σ 1/(k + rank_i) for each source
```

Benefits:
- No need to normalize scores between sources
- Robust to outliers
- Simple and effective
