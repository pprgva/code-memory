# /search - Semantic code search

Search the codebase using natural language with grepai.

## Usage

Argument: the search query in English.

```bash
grepai search "$ARGUMENTS" --json --compact
```

If grepai is not available or fails, fall back to Grep/Glob tools.
