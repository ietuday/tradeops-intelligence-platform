# Secrets Management

The chart supports three modes:

- Existing Kubernetes Secret: recommended for staging and production.
- Inline development Secret: local-only, explicitly enabled through `secrets.allowInlineValues`.
- External Secrets Operator: optional example in `deployments/helm/tradeops/examples/external-secret.yaml`.

Expected secret keys:

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `IDENTITY_JWT_SECRET`
- `IDENTITY_REFRESH_TOKEN_SECRET`
- `GRAFANA_ADMIN_PASSWORD` when Grafana is managed by the deployment

AWS Secrets Manager, Azure Key Vault, Google Secret Manager, and HashiCorp Vault can all feed the Kubernetes Secret through External Secrets Operator or Secrets Store CSI Driver. The primary chart does not require those CRDs unless the optional integration is enabled.

