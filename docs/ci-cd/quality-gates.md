# CI/CD Quality Gates

TradeOps uses GitHub Actions and local Makefile commands to keep the repository testable, demo-ready, and interview-explainable. The workflows are intentionally practical for a portfolio platform: they validate code, scripts, Docker images, docs, and common security concerns without requiring a full production infrastructure environment.

## Workflow Overview

| Workflow | File | Purpose |
| --- | --- | --- |
| CI | `.github/workflows/ci.yml` | Runs API Gateway tests, Go service tests, Python service tests, script syntax checks, and Docker Compose config validation. |
| Security | `.github/workflows/security.yml` | Runs secret scanning, Go vet/govulncheck, Node audit, optional Python dependency audit, and a basic secret-pattern grep. |
| Docker | `.github/workflows/docker.yml` | Builds all service Docker images locally with `:ci` tags. Images are not pushed. |
| Docs | `.github/workflows/docs.yml` | Ensures required documentation exists and runs markdown linting as a non-blocking review aid. |

## What Runs On Pull Requests

- API Gateway `npm ci`, Jest tests, build, and lint when scripts exist.
- Go module download and `go test ./...` for Go services.
- Python dependency install and `pytest` when Python tests exist.
- Bash syntax validation for smoke and demo scripts.
- Docker Compose configuration validation.
- Security checks and dependency audits.
- Docker image build validation.
- Required documentation checks.

## What Runs On Push To Main

The same quality gates run on pushes to `main` so the default branch stays release-ready.

The security and Docker workflows also support `workflow_dispatch` for manual runs.

## Test Strategy

- Unit and service-level tests are run inside each service directory.
- API Gateway proxy tests run with Jest.
- Go services run `go test ./...`.
- Python services run `pytest` only when tests are present.
- Full platform smoke tests remain local/manual because they require the Docker Compose stack to be running.

## Docker Build Validation

The Docker workflow builds images for:

- `tradeops/api-gateway:ci`
- `tradeops/identity-service:ci`
- `tradeops/market-data-service:ci`
- `tradeops/order-service:ci`
- `tradeops/portfolio-service:ci`
- `tradeops/surveillance-service:ci`
- `tradeops/notification-service:ci`
- `tradeops/strategy-service:ci`
- `tradeops/risk-engine-service:ci`

Images are built for validation only and are not pushed to a registry.

## Security Scanning

Security automation includes:

- Gitleaks secret scanning as a review aid.
- Basic grep for suspicious assignments such as `JWT_SECRET=`, `PASSWORD=`, `API_KEY=`, and `PRIVATE_KEY=`.
- `go vet` for Go services.
- `govulncheck` for Go vulnerability scanning.
- `npm audit --audit-level=high` for API Gateway dependencies.
- `pip-audit` for Python services as an optional/non-blocking check.

The security workflow is useful but intentionally not over-tuned. Some local demo placeholders may need human review rather than automatic failure.

## Known Limitations

- Workflows do not start the full Docker Compose platform.
- End-to-end smoke tests are local/manual because they need running infrastructure.
- Python dependency audits can be noisy, so they are non-blocking by default.
- Markdown linting is non-blocking and meant to catch obvious documentation issues.
- No images are pushed to a container registry.
- No Kubernetes, Helm, or cloud deployment validation is included.

## Run Locally

Use the Makefile for local quality checks:

```bash
make help
make test-node
make test-go
make test-python
make validate-scripts
make compose-config
make docker-build
```

Run demo and smoke commands against a running stack:

```bash
make dev-up
make smoke
make demo-surveillance
make demo-notifications
make demo-e2e
make dev-down
```
