SHELL := /bin/sh

.PHONY: help fmt test vet check run-api run-worker web-install run-web web-build compose-up compose-down

help:
	@printf '%s\n' \
	  'make check        Format-check, test and vet Go code' \
	  'make run-api      Run the API on :8080' \
	  'make run-worker   Run the durable-worker scaffold' \
	  'make web-install  Install web dependencies' \
	  'make run-web      Run the Vite client on :5173' \
	  'make web-build    Type-check and build the client' \
	  'make compose-up   Start PostgreSQL, API and web services'

fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

test:
	go test ./...

vet:
	go vet ./...

check:
	@unformatted="$$(gofmt -l $$(find cmd internal -name '*.go' -type f))"; \
	if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
	go test ./...
	go vet ./...

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

web-install:
	npm --prefix web install --no-audit --no-fund

run-web:
	npm --prefix web run dev -- --host 0.0.0.0

web-build:
	npm --prefix web run typecheck
	npm --prefix web run build

compose-up:
	docker compose up --build

compose-down:
	docker compose down -v
