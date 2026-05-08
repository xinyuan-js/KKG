SHELL := /bin/zsh
WEB_NODE_MODULES_VOL ?= blog_web_node_modules
WEB_NEXT_VOL ?= blog_web_next

.PHONY: env-init dev-core dev-pro dev-search dev-observe dev-stream dev-gateway dev-oj dev-all stop logs ps clean api-build api-test api-run web-install web-build web-run web-dev web-stop web-reset gateway-run gateway-stop oj-dev oj-run oj-stop oj-logs
.PHONY: api-dev api-stop api-logs
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

api-run:
	@ids=$$(docker ps -q --filter publish=8080); if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null 2>&1 || true; fi
	docker run --rm --name blog-api-run -v $(PWD)/kkg-blog-backend:/src -w /src --env-file .env -e MYSQL_HOST=host.docker.internal -e REDIS_HOST=host.docker.internal -e MINIO_ENDPOINT=host.docker.internal:9000 -e MINIO_PUBLIC_BASE_URL=http://127.0.0.1:9000 -e ELASTICSEARCH_URL=http://host.docker.internal:9200 -e SERVER_PORT=8080 -p 8080:8080 golang:1.22-bookworm sh -lc '/usr/local/go/bin/go run ./cmd/api'

api-dev:
	docker rm -f blog-api-dev >/dev/null 2>&1 || true
	@ids=$$(docker ps -q --filter publish=8080); if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null 2>&1 || true; fi
	docker run -d --name blog-api-dev -v $(PWD)/kkg-blog-backend:/app -w /app --env-file .env -e MYSQL_HOST=host.docker.internal -e REDIS_HOST=host.docker.internal -e MINIO_ENDPOINT=host.docker.internal:9000 -e MINIO_PUBLIC_BASE_URL=http://127.0.0.1:9000 -e ELASTICSEARCH_URL=http://host.docker.internal:9200 -e SERVER_PORT=8080 -p 8080:8080 cosmtrek/air -c .air.toml
	@echo "api dev server started with hot reload: http://127.0.0.1:8080"

api-stop:
	docker rm -f blog-api-dev >/dev/null 2>&1 || true

api-logs:
	docker logs -f blog-api-dev

web-install:
	docker run --rm -v $(PWD)/apps/web:/app -v $(WEB_NODE_MODULES_VOL):/app/node_modules -w /app node:20-bookworm sh -lc 'npm install'

web-build:
	docker run --rm -v $(PWD)/apps/web:/app -v $(WEB_NODE_MODULES_VOL):/app/node_modules -v $(WEB_NEXT_VOL):/app/.next -w /app node:20-bookworm sh -lc 'npm run build'

web-dev:
	docker rm -f blog-web-dev >/dev/null 2>&1 || true
	@ids=$$(docker ps -q --filter publish=3001); if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null 2>&1 || true; fi
	docker run -d --name blog-web-dev -v $(PWD)/apps/web:/app -v $(WEB_NODE_MODULES_VOL):/app/node_modules -v $(WEB_NEXT_VOL):/app/.next -w /app -e NEXT_PUBLIC_API_BASE=/blog-api -e API_BASE_SERVER=http://host.docker.internal:8080 -e NEXT_PUBLIC_OJ_API_BASE=/oj-api -e OJ_API_BASE_SERVER=http://host.docker.internal:8121 -e WATCHPACK_POLLING=true -e CHOKIDAR_USEPOLLING=true -p 3001:3001 node:20-bookworm sh -lc 'npm install && rm -rf .next/* && npm run dev'
	@echo "web dev server started: http://127.0.0.1:3001"

web-run:
	docker rm -f blog-web-dev >/dev/null 2>&1 || true
	@ids=$$(docker ps -q --filter publish=3001); if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null 2>&1 || true; fi
	docker run -d --name blog-web-dev -v $(PWD)/apps/web:/app -v $(WEB_NODE_MODULES_VOL):/app/node_modules -v $(WEB_NEXT_VOL):/app/.next -w /app -e NEXT_PUBLIC_API_BASE=/blog-api -e API_BASE_SERVER=http://host.docker.internal:8080 -e NEXT_PUBLIC_OJ_API_BASE=/oj-api -e OJ_API_BASE_SERVER=http://host.docker.internal:8121 -p 3001:3001 node:20-bookworm sh -lc 'npm install && rm -rf .next/* && npm run build && npm run start'
	@echo "web server started (stable mode): http://127.0.0.1:3001"

web-stop:
	docker rm -f blog-web-dev >/dev/null 2>&1 || true

web-reset:
	docker rm -f blog-web-dev >/dev/null 2>&1 || true
	rm -rf apps/web/.next
	docker volume rm $(WEB_NODE_MODULES_VOL) >/dev/null 2>&1 || true
	docker volume rm $(WEB_NEXT_VOL) >/dev/null 2>&1 || true
	@echo "web cache reset done"

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

oj-run:
	docker rm -f oj-api-dev >/dev/null 2>&1 || true
	@ids=$$(docker ps -q --filter publish=8121); if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null 2>&1 || true; fi
	docker run -d --name oj-api-dev -v $(PWD)/kkg-oj-backend:/app -w /app -e SERVER_PORT=8121 -e MYSQL_DSN="root:$$(grep MYSQL_ROOT_PASSWORD .env | cut -d= -f2)@tcp(host.docker.internal:$$(grep MYSQL_PORT .env | cut -d= -f2))/$$(grep MYSQL_DATABASE .env | cut -d= -f2)?charset=utf8mb4&parseTime=True&loc=Local" -e REDIS_ADDR="host.docker.internal:$$(grep REDIS_PORT .env | cut -d= -f2)" -e REDIS_PASSWORD="$$(grep REDIS_PASSWORD .env | cut -d= -f2)" -e REDIS_DB=1 -e JWT_SECRET="$$(grep JWT_SECRET .env | cut -d= -f2)" -e RABBITMQ_ENABLED=true -e RABBITMQ_URL="amqp://$$(grep RABBITMQ_USER .env | cut -d= -f2):$$(grep RABBITMQ_PASSWORD .env | cut -d= -f2)@blog-rabbitmq:5672/" -e RABBITMQ_JUDGE_QUEUE=oj.judge -e JUDGE_SANDBOX_TYPE=remote -e JUDGE_SANDBOX_URL=http://host.docker.internal:8082/executeCode -e JUDGE_AUTH_SECRET=secretKey -p 8121:8121 golang:1.23-bookworm sh -lc '/usr/local/go/bin/go run ./cmd/server'
	@echo "oj api started: http://127.0.0.1:8121"

oj-stop:
	docker rm -f oj-api-dev >/dev/null 2>&1 || true

oj-logs:
	docker logs -f oj-api-dev
