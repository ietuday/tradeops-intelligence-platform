DOCKER_COMPOSE_FILE=infrastructure/docker/docker-compose.yml

.PHONY: dev-up dev-down logs ps test smoke

dev-up:
	docker compose -f $(DOCKER_COMPOSE_FILE) up --build -d

dev-down:
	docker compose -f $(DOCKER_COMPOSE_FILE) down

logs:
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f

ps:
	docker compose -f $(DOCKER_COMPOSE_FILE) ps

test:
	cd services/api-gateway && npm test

smoke:
	./scripts/smoke-test.sh