# grepai - Semantic Code Search

> **MANDATORY RULE**: This project uses grepai for ALL code searches.
> Do NOT use grep, rg, find, glob, or any other search tool.
> A hook will block these attempts automatically.

## First Step - Always Run This

```bash
grepai init
```

This returns JSON telling you exactly what to do:
- `status: ready` → Use the commands below
- `status: needs_setup` → Run the `next_step` command provided
- `status: needs_machine_setup` → Ask user to run `grepai machine setup`

## Quick Reference

| Task | Command |
|------|---------|
| Initialize | `grepai init` |
| Search code | `grepai search "query in english" --json --compact` |
| Find callers | `grepai trace callers "Symbol" --json` |
| Find callees | `grepai trace callees "Symbol" --json` |
| Call graph | `grepai trace graph "Symbol" --depth 3 --json` |
| Health check | `grepai doctor` |

## CRITICAL RULES

1. **ALWAYS use English** for search queries (embedding model is English-trained)
2. **ALWAYS use `--json --compact`** for programmatic consumption (saves tokens)
3. **NEVER use**: `grep`, `rg`, `ripgrep`, `ag`, `ack`, `find -name`, `glob`
4. **Describe intent**, not code: "handles user login" NOT "func Login"

## Search Examples

```bash
# Semantic search
grepai search "error handling middleware" --json --compact
grepai search "database connection pool" --json --compact
grepai search "JWT token validation" --limit 5 --json
```

## Trace Examples (Call Graph Analysis)

```bash
# Find all callers of a function
grepai trace callers "HandleRequest" --json

# Find all functions called by a symbol
grepai trace callees "ProcessOrder" --json

# Full call graph around a symbol
grepai trace graph "ValidateToken" --depth 3 --json
```

## When grepai Fails

1. Run `grepai init` to check status
2. Run `grepai doctor` to diagnose
3. If truly broken: use grep/glob as temporary fallback and report to user

## Configuration

- Per-project: `.grepai/config.yaml`
- Machine-level: `~/.grepai/machine.yaml`

---

## Hook Protection

This project includes a PreToolUse hook that:
- Blocks `grep`, `rg`, `ag`, `find -name`, and similar commands
- Blocks `glob` tool usage
- Provides guidance to use grepai instead

**The hook cannot be bypassed.**
