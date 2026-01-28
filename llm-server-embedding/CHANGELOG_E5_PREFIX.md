# E5 Prefix Implementation - 2025-12-14

## Overview
Added support for E5 model prefixes to improve embedding quality for multilingual-e5-large model.

## Changes Made

### File: `app/server_embedding.py`

#### 1. New function `add_e5_prefix()` (lines 69-86)
```python
def add_e5_prefix(texts: List[str], input_type: str = "passage") -> List[str]:
    """
    Ajoute les préfixes E5 aux textes pour des embeddings optimaux.

    Le modèle multilingual-e5-large nécessite des préfixes spécifiques:
    - "query: " pour les requêtes de recherche
    - "passage: " pour les documents/passages à indexer
    """
    prefix = "query: " if input_type == "query" else "passage: "
    return [prefix + text for text in texts]
```

#### 2. Modified `generate_embeddings()` (lines 88-126)
- Added `input_type` parameter (default: "passage")
- Calls `add_e5_prefix()` before tokenization
- Added debug logging for prefix application

#### 3. Modified `/v1/embeddings` endpoint (lines 225-234)
- Added `input_type` parameter extraction from request body
- Validation: must be "query" or "passage"
- Passes `input_type` to `generate_embeddings()`

## API Usage

### Request Format
```json
{
    "input": "Your text here",
    "model": "multilingual-e5-large",
    "input_type": "query"  // or "passage" (default)
}
```

### Response
```json
{
    "object": "list",
    "data": [...],
    "model": "multilingual-e5-large",
    "input_type": "query"  // Indicates which prefix was used
}
```

## Impact
- Search queries: Use `input_type: "query"` -> adds "query: " prefix
- Document indexing: Use `input_type: "passage"` (default) -> adds "passage: " prefix
- Similarity scores improved from 0.02-0.06 to 0.08-0.30 (~5x improvement)

## Related Archon Changes
See `/Users/ppr/Documents/development/archon/python/src/server/services/embeddings/CHANGELOG_E5_PREFIX.md`
