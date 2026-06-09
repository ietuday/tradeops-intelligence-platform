# Secrets Management

TradeOps keeps secret handling simple for local demos and explicit for production gaps.

## Rules

- Never commit real `.env` files.
- Use `.env.example` files for names and placeholders only.
- Keep `infrastructure/docker/.env` local and ignored.
- Do not put real JWT secrets, database passwords, API keys, private keys, webhook secrets, or SMTP credentials in docs or scripts.

## Local Docker Compose

```bash
cp infrastructure/docker/.env.example infrastructure/docker/.env
```

Use long random values for JWT secrets if sharing a demo outside your machine. Local defaults and placeholders are not production credentials.

## Helm And Kubernetes

The Helm chart includes an example Secret placeholder so the chart can render. For real clusters:

- Create Kubernetes Secrets through a controlled process.
- Keep Secret manifests out of public repositories.
- Prefer External Secrets, Vault, or a cloud secret manager.
- Rotate secrets after demos, incidents, and contributor access changes.

## Specific Guidance

| Secret Type | Guidance |
| --- | --- |
| JWT secrets | Use high-entropy values; rotate by issuing new tokens and expiring old sessions. |
| Database credentials | Use per-environment credentials; avoid sharing admin credentials across services in production. |
| Webhook URLs | Treat as sensitive because they can disclose internal endpoints or tokens. |
| API keys/private keys | Never commit; scan before release. |

## Checks

```bash
./scripts/security-check.sh
git status --short
git ls-files | grep -E '(^|/)(\.env|.*\.pem|.*\.key|id_rsa|private_key)$'
```

The script is read-only and designed to flag risky committed files and secret-like strings for review.
