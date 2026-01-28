# grepai - Semantic Code Search

## What is grepai

grepai is a CLI tool installed at `~/.local/bin/grepai`. It indexes code using vector embeddings (E5 model) for natural language search. It runs locally, no cloud.

## Binary Location

```
/Users/ppr/.local/bin/grepai
```

## Prerequisites

Before using grepai in a project:

1. The project must be initialized: `grepai init` (creates `.grepai/config.yaml`)
2. The watcher must have run at least once: `grepai watch` (indexes files)
3. Check health: `grepai doctor`

## Commands Reference

### Search (primary use case)

```bash
# Natural language search - ALWAYS use English queries for best results
grepai search "error handling logic"
grepai search "database connection" --limit 5

# JSON output for programmatic use
grepai search "authentication flow" --json

# Compact JSON (saves tokens, omits content)
grepai search "validation" --json --compact
```

### Trace (call graph analysis)

```bash
# Find all callers of a function
grepai trace callers "HandleRequest" --json

# Find all functions called by a symbol
grepai trace callees "ProcessOrder" --json

# Full call graph
grepai trace graph "ValidateToken" --depth 3 --json
```

### Doctor (health check)

```bash
grepai doctor        # Human-readable output
grepai doctor --json # JSON output
```

### Watch (indexing)

```bash
grepai watch                # Foreground (Ctrl+C to stop)
grepai watch --background   # Background daemon
grepai watch --status       # Check if running
grepai watch --stop         # Stop daemon
```

### Other

```bash
grepai init                          # Initialize project
grepai init --yes                    # Non-interactive init
grepai init --backend postgres       # With postgres backend
grepai status                        # Index statistics
grepai workspace                     # Multi-project management
```

## When to Use grepai vs Grep/Glob

**Use grepai search for:**
- Understanding what code does or where functionality lives
- Finding implementations by intent ("authentication logic", "error handling")
- Exploring unfamiliar parts of the codebase
- Any search where you describe WHAT the code does

**Use standard Grep/Glob for:**
- Exact text matching (variable names, imports, specific strings)
- File path patterns (`**/*.go`)

**Fallback:** If grepai fails (not running, index unavailable), fall back to Grep/Glob.

## Query Tips

- **Use English** for queries (embedding model is English-trained)
- **Describe intent**, not implementation: "handles user login" not "func Login"
- **Be specific**: "JWT token validation" better than "token"

## Configuration

Per-project config in `.grepai/config.yaml`. Key settings:
- `store.backend`: "gob" (file-based), "postgres", or "qdrant"
- `chunking.size`: 512 (token chunk size)
- `chunking.overlap`: 100 (overlap between chunks)
- `embedder.model_path`: path to E5 model directory

Global resources in `~/.grepai/`:
- `venv/` - Python virtual environment (torch + transformers)
- `models/multilingual-e5-large/` - E5 embedding model
