# Skill: grepai search

## Trigger
Use this skill when you need to explore or search code in a project that has been initialized with grepai (`.grepai/config.yaml` exists).

## Usage

```bash
# Semantic search - always use English queries
grepai search "<natural language query>"

# For AI agent consumption
grepai search "<query>" --json --compact

# Limit results
grepai search "<query>" --limit 5
```

## Examples

```bash
grepai search "how are errors handled"
grepai search "database migration logic" --json --compact
grepai search "API authentication middleware" --limit 3
```

## Rules

1. ALWAYS use English queries (the embedding model is English-trained)
2. Describe intent, not code: "handles user login" NOT "func Login"
3. Use `--json --compact` when processing results programmatically
4. If grepai fails, fall back to Grep/Glob tools
5. The project must have `.grepai/config.yaml` — if not, run `grepai init --yes` first
