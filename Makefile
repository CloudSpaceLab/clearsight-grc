SHELL := /bin/sh
.PHONY: check test test-postgres test-integration vet format run-api run-api-postgres run-worker run-worker-postgres run-ai-gateway web-install run-web compose-up compose-down
check: format test vet
format:
	@test -z "$$(gofmt -l $$(find cmd internal -name '*.go' -type f))" || (gofmt -w $$(find cmd internal -name '*.go' -type f); exit 1)
test:
	go test ./...
test-postgres:
	go test -tags postgres ./...
test-integration:
	go test -tags "postgres postgresintegration" ./internal/integration
vet:
	go vet ./...
run-api:
	go run ./cmd/api
run-api-postgres:
	go run -tags postgres ./cmd/api
run-worker:
	go run ./cmd/worker
run-worker-postgres:
	go run -tags postgres ./cmd/worker
run-ai-gateway:
	go run ./cmd/ai-gateway
web-install:
	cd web && npm install --no-audit --no-fund
run-web:
	cd web && npm run dev
compose-up:
	docker compose up --build
compose-down:
	docker compose down -v
