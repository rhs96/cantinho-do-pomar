COMPOSE := docker compose -f deploy/compose.yaml

.PHONY: up down logs ps api-test api-lint

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

api-test:
	cd services/api && go test -race ./...

api-lint:
	cd services/api && golangci-lint run

.PHONY: web-dev web-test web-lint

web-dev:
	cd apps/web && npm run dev

web-test:
	cd apps/web && npm test

web-lint:
	cd apps/web && npm run lint && npm run typecheck
