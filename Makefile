SHELL := /bin/zsh

.PHONY: env-init dev-core dev-pro dev-search dev-observe dev-stream dev-gateway dev-oj dev-all stop logs ps clean api-build api-test gateway-run gateway-stop oj-dev
.PHONY: sandbox-dev sandbox-rebuild sandbox-stop sandbox-logs oj-api-dev oj-api-restart oj-api-stop oj-api-logs

env-init:
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@echo ".env ready"

dev-core: env-init
	docker compose up -d mysql redis minio
	@echo "core services started: mysql redis minio"

dev-pro: env-init
	docker compose --profile pro up -d
	@echo "pro profile started (includes rabbitmq + core services)"

dev-search: env-init
	docker compose --profile search up -d
	@echo "search profile started (elasticsearch + kibana + core services)"

dev-observe: env-init
	docker compose --profile observe up -d
	@echo "observe profile started (jaeger + prometheus + grafana + core services)"

dev-stream: env-init
	docker compose --profile stream up -d
	@echo "stream profile started (zookeeper + kafka + core services)"

dev-gateway: env-init
	docker compose --profile gateway up -d
	@echo "gateway profile started (nginx on :80)"

dev-oj: env-init
	docker compose --profile oj up -d oj-sandbox oj-api
	@echo "oj profile started (oj-api on :8121)"

dev-all: env-init
	docker compose --profile pro --profile search --profile observe --profile stream --profile gateway --profile oj up -d
	@echo "all profiles started"

stop:
	docker compose down

logs:
	docker compose logs -f --tail=100

ps:
	docker compose ps

clean:
	docker compose down -v

api-build:
	docker run --rm -v $(PWD)/kkg-blog-backend:/src -w /src golang:1.22-bookworm sh -lc '/usr/local/go/bin/go build ./cmd/api'

api-test:
	docker run --rm -v $(PWD)/kkg-blog-backend:/src -w /src golang:1.22-bookworm sh -lc '/usr/local/go/bin/go test ./...'

gateway-run:
	docker compose --profile gateway up -d nginx
	@echo "gateway started: http://127.0.0.1"

gateway-stop:
	docker rm -f kkg-gateway >/dev/null 2>&1 || true

oj-dev:
	docker compose --profile oj up -d oj-sandbox oj-api
	@echo "oj api started: http://127.0.0.1:8121"

oj-api-dev:
	docker compose --profile oj up -d oj-api
	@echo "oj api only started: http://127.0.0.1:8121"

oj-api-restart:
	docker compose restart oj-api

oj-api-stop:
	docker compose stop oj-api

oj-api-logs:
	docker compose logs -f --tail=100 oj-api

sandbox-dev:
	docker compose --profile oj up -d oj-sandbox
	@echo "sandbox only started: http://127.0.0.1:8082"

sandbox-rebuild:
	docker compose --profile oj up -d --build oj-sandbox
	@echo "sandbox rebuilt and restarted"

sandbox-stop:
	docker compose stop oj-sandbox

sandbox-logs:
	docker compose logs -f --tail=100 oj-sandbox
