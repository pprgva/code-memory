---
title: Search Boost
description: Improve search relevance with structural boosting
---

Structural boosting automatically adjusts search scores based on file paths. Test files are penalized, source directories are boosted.

## How It Works

After vector similarity search, grepai applies multipliers to scores:

- **Penalties** (`factor < 1.0`): Reduce score for tests, mocks, generated code
- **Bonuses** (`factor > 1.0`): Increase score for source directories
- **Cumulative**: Multiple matching patterns are multiplied together

Example: `tests/auth_test.py` matches `/tests/` (×0.5) and `_test.` (×0.5) → final factor = 0.25

## Default Configuration

Enabled by default with language-agnostic patterns:

```yaml
search:
  boost:
    enabled: true
    penalties:
      # Test files (multi-language)
      - pattern: "/tests/"
        factor: 0.5
      - pattern: "/test/"
        factor: 0.5
      - pattern: "__tests__"
        factor: 0.5
      - pattern: "_test."
        factor: 0.5
      - pattern: ".test."
        factor: 0.5
      - pattern: ".spec."
        factor: 0.5
      - pattern: "test_"
        factor: 0.5
      # Mocks
      - pattern: "/mocks/"
        factor: 0.4
      - pattern: "/mock/"
        factor: 0.4
      - pattern: ".mock."
        factor: 0.4
      # Fixtures & test data
      - pattern: "/fixtures/"
        factor: 0.4
      - pattern: "/testdata/"
        factor: 0.4
      # Generated code
      - pattern: "/generated/"
        factor: 0.4
      - pattern: ".generated."
        factor: 0.4
      - pattern: ".gen."
        factor: 0.4
      # Documentation
      - pattern: ".md"
        factor: 0.6
      - pattern: "/docs/"
        factor: 0.6
    bonuses:
      # Source directories (multi-language)
      - pattern: "/src/"
        factor: 1.1
      - pattern: "/lib/"
        factor: 1.1
      - pattern: "/app/"
        factor: 1.1
```

## Default Rules Summary

| Category | Patterns | Factor |
|----------|----------|--------|
| Tests | `/tests/`, `/test/`, `__tests__`, `_test.`, `.test.`, `.spec.`, `test_` | ×0.5 |
| Mocks | `/mocks/`, `/mock/`, `.mock.` | ×0.4 |
| Fixtures | `/fixtures/`, `/testdata/` | ×0.4 |
| Generated | `/generated/`, `.generated.`, `.gen.` | ×0.4 |
| Docs | `.md`, `/docs/` | ×0.6 |
| Source | `/src/`, `/lib/`, `/app/` | ×1.1 |

## Customization

### Add custom patterns

```yaml
search:
  boost:
    enabled: true
    penalties:
      - pattern: "/vendor/"
        factor: 0.3
      - pattern: ".min.js"
        factor: 0.2
    bonuses:
      - pattern: "/core/"
        factor: 1.2
```

### Disable boosting

```yaml
search:
  boost:
    enabled: false
```

## Pattern Matching

Patterns use simple substring matching on file paths:

- `/tests/` matches `project/tests/unit/auth.py`
- `_test.` matches `auth_test.go`, `user_test.py`
- `.spec.` matches `auth.spec.ts`, `user.spec.js`

Use `/` to delimit directories and avoid false positives (e.g., `/tests/` won't match `contests/`).
