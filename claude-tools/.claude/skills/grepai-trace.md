# Skill: grepai trace

## Trigger
Use this skill when you need to understand function relationships: who calls what, what gets called by what.

## Usage

```bash
# Find all callers of a symbol
grepai trace callers "<symbol>" --json

# Find all callees of a symbol
grepai trace callees "<symbol>" --json

# Full call graph around a symbol
grepai trace graph "<symbol>" --depth 3 --json
```

## When to Use

- Before modifying a function: find all callers to assess impact
- Understanding data flow: trace callees to see what a function depends on
- Exploring architecture: build a graph around key symbols

## Rules

1. Always use `--json` for programmatic consumption
2. Symbol names are case-sensitive
3. The project must be initialized and indexed with `grepai watch`
