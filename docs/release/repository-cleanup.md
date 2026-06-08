# Repository Cleanup Guide

Use this guide before creating a public GitHub release or sharing the repository. These commands remove local/generated files only. Review each command before running it.

## Check What Would Be Removed

```bash
git status --short
find . -name node_modules -type d -prune
find . -name dist -type d -prune
find . -name .venv -type d -prune
find . -name __pycache__ -type d -prune
find . -name .pytest_cache -type d -prune
find . -name '*.log' -type f
find . -name .env -type f
```

## Remove JavaScript Generated Files

```bash
find . -name node_modules -type d -prune -exec rm -rf {} +
find . -name dist -type d -prune -exec rm -rf {} +
find . -name coverage -type d -prune -exec rm -rf {} +
```

## Remove Python Generated Files

```bash
find . -name .venv -type d -prune -exec rm -rf {} +
find . -name __pycache__ -type d -prune -exec rm -rf {} +
find . -name .pytest_cache -type d -prune -exec rm -rf {} +
```

## Remove Local Environment Files

Do not remove `.env.example` files.

```bash
find . -name .env -type f -print
rm -f .env
rm -f infrastructure/docker/.env
```

## Remove Logs And Temporary Files

```bash
find . -name '*.log' -type f -delete
rm -rf tmp/
```

## Final Verification

```bash
git status --short
git check-ignore node_modules dist .venv __pycache__ .pytest_cache coverage tmp .env infrastructure/docker/.env
git check-ignore -v infrastructure/docker/.env
```

Expected tracked release files that should remain:

- `.env.example` files
- `go.sum`
- `package-lock.json`
- `requirements.txt`
- `docs/`
- `scripts/`
- `migrations/`
