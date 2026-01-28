# /doctor - Check grepai health

Run `grepai doctor` to verify the installation is working correctly.

```bash
grepai doctor
```

If any check fails, follow the hint provided in the output.

Common fixes:
- Config FAIL → `grepai init --yes`
- Venv FAIL → `grepai init --yes`
- Store FAIL → Check database connection / Docker is running
- Index empty → `grepai watch`
