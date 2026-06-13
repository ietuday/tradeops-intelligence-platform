# Database Migrations

The Helm chart includes an idempotent migration Job that applies the existing SQL migration set from chart-bundled files.

Local values enable migrations. Demo seed data is enabled only by `values-local.yaml`; staging and production disable seeds.

Helm hooks are supported but disabled by default. Hooks can simplify install ordering, but they also affect rollback and lifecycle behavior. For production, review migration jobs separately and back up the database before upgrades.

Rollback note: database migrations are forward-only unless explicit down migrations exist and are tested.

