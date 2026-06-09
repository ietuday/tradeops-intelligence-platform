DOCKER_COMPOSE_FILE := infrastructure/docker/docker-compose.yml
DOCKER_ENV_FILE := infrastructure/docker/.env
DOCKER_ENV_EXAMPLE_FILE := infrastructure/docker/.env.example
HELM_CHART := infrastructure/helm/tradeops-platform

GO_SERVICES := identity-service market-data-service order-service portfolio-service surveillance-service notification-service
PYTHON_SERVICES := strategy-service risk-engine-service

.PHONY: help test test-go test-node test-python compose-config validate-scripts security-check perf-smoke load-test-gateway load-test-all helm-lint helm-template validate-helm smoke demo-surveillance demo-notifications demo-e2e docker-build clean clean-local dev-up dev-down logs ps

help:
	@echo "TradeOps local commands"
	@echo ""
	@echo "  make test                 Run Node, Go, and Python tests"
	@echo "  make test-go              Run Go service tests"
	@echo "  make test-node            Run API Gateway tests"
	@echo "  make test-python          Run Python service tests when tests exist"
	@echo "  make compose-config       Validate Docker Compose config"
	@echo "  make validate-scripts     Validate Bash script syntax"
	@echo "  make security-check       Run read-only local security checks"
	@echo "  make perf-smoke           Run lightweight curl timing checks"
	@echo "  make load-test-gateway    Run optional k6 gateway load test"
	@echo "  make load-test-all        Run all optional k6 load tests with low defaults"
	@echo "  make validate-helm        Validate optional Helm chart when Helm is installed"
	@echo "  make smoke                Run smoke test against a running stack"
	@echo "  make demo-surveillance    Run surveillance demo"
	@echo "  make demo-notifications   Run notification demo"
	@echo "  make demo-e2e             Run end-to-end demo"
	@echo "  make docker-build         Build service Docker images locally"
	@echo "  make clean                Explain cleanup options"
	@echo "  make clean-local          Remove local/generated folders"
	@echo "  make dev-up               Start local Docker Compose stack"
	@echo "  make dev-down             Stop local Docker Compose stack"

test: test-node test-go test-python

test-node:
	cd services/api-gateway && npm test -- --runInBand

test-go:
	@for service in $(GO_SERVICES); do \
		echo "==> go test services/$$service"; \
		(cd services/$$service && go test ./...) || exit $$?; \
	done

test-python:
	@for service in $(PYTHON_SERVICES); do \
		echo "==> pytest services/$$service"; \
		if [ -d services/$$service/tests ]; then \
			(cd services/$$service && python -m pytest) || exit $$?; \
		else \
			echo "No tests directory for $$service; skipping."; \
		fi; \
	done

compose-config:
	docker compose --env-file $(DOCKER_ENV_EXAMPLE_FILE) -f $(DOCKER_COMPOSE_FILE) config

validate-scripts:
	bash -n scripts/security-check.sh
	bash -n scripts/run-load-tests.sh
	bash -n scripts/perf-smoke.sh
	bash -n scripts/smoke-test.sh
	bash -n scripts/demo-surveillance.sh
	bash -n scripts/demo-notifications.sh
	bash -n scripts/demo-e2e-tradeops.sh
	bash -n scripts/demo-reliability.sh
	bash -n scripts/demo-audit.sh
	bash -n scripts/demo-observability.sh
	bash -n scripts/demo-correlation-tracing.sh
	bash -n scripts/demo-websocket-streams.sh
	bash -n scripts/demo-otel-tracing.sh
	bash -n scripts/db-migrate.sh
	bash -n scripts/db-seed.sh
	bash -n scripts/demo-db-migrations.sh
	bash -n scripts/db-backup.sh
	bash -n scripts/db-restore.sh
	bash -n scripts/archive-old-data.sh
	bash -n scripts/replay-sample-events.sh
	bash -n scripts/replay-dlq-events.sh
	bash -n scripts/validate-helm.sh

security-check:
	./scripts/security-check.sh

perf-smoke:
	./scripts/perf-smoke.sh

load-test-gateway:
	./scripts/run-load-tests.sh --gateway

load-test-all:
	./scripts/run-load-tests.sh --all

helm-lint:
	helm lint $(HELM_CHART)

helm-template:
	helm template tradeops $(HELM_CHART)

validate-helm:
	./scripts/validate-helm.sh

smoke:
	./scripts/smoke-test.sh

demo-surveillance:
	./scripts/demo-surveillance.sh

demo-notifications:
	./scripts/demo-notifications.sh

demo-e2e:
	./scripts/demo-e2e-tradeops.sh

docker-build:
	docker build -t tradeops/api-gateway:ci services/api-gateway
	docker build -t tradeops/identity-service:ci services/identity-service
	docker build -t tradeops/market-data-service:ci services/market-data-service
	docker build -t tradeops/order-service:ci services/order-service
	docker build -t tradeops/portfolio-service:ci services/portfolio-service
	docker build -t tradeops/surveillance-service:ci services/surveillance-service
	docker build -t tradeops/notification-service:ci services/notification-service
	docker build -t tradeops/strategy-service:ci services/strategy-service
	docker build -t tradeops/risk-engine-service:ci services/risk-engine-service

clean:
	@echo "No files removed. Use 'make clean-local' to remove generated local folders."
	@echo "See docs/release/repository-cleanup.md before publishing."

clean-local:
	find . -name node_modules -type d -prune -exec rm -rf {} +
	find . -name dist -type d -prune -exec rm -rf {} +
	find . -name coverage -type d -prune -exec rm -rf {} +
	find . -name .venv -type d -prune -exec rm -rf {} +
	find . -name __pycache__ -type d -prune -exec rm -rf {} +
	find . -name .pytest_cache -type d -prune -exec rm -rf {} +
	find . -name '*.log' -type f -delete

dev-up:
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) up --build -d

dev-down:
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) down

logs:
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) logs -f

ps:
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) ps
