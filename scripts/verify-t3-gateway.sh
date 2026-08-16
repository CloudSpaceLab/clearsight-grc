#!/usr/bin/env bash
set -Eeuo pipefail

required=(
  cmd/ai-gateway/main.go
  internal/aigateway/types.go
  internal/aigateway/service.go
  internal/aigateway/routes.go
  internal/aigateway/provider_openai.go
  internal/aigateway/provider_anthropic.go
  internal/aigateway/stream_sinks.go
  internal/aigateway/telemetry.go
  api/ai-gateway.openapi.json
  Dockerfile.ai-gateway
  deploy/ai-gateway.config.example.json
  docs/architecture/ai-gateway-transport.md
  docs/acceptance/t3-ai-gateway-transport.md
)
for file in "${required[@]}"; do
  test -s "$file" || { echo "missing T3 gateway contract: $file" >&2; exit 1; }
done

if find migrations -maxdepth 1 -type f -name '*ai_gateway*' -print -quit | grep -q .; then
  echo "T3 transport introduced durable gateway persistence" >&2
  exit 1
fi
if grep -R -nE --include='*.go' 'internal/(sourceaccess|authority|governance|workflow|continuity)' internal/aigateway cmd/ai-gateway; then
  echo "T3 transport depends on governed enforcement or product-domain packages" >&2
  exit 1
fi
if grep -R -nE --include='*.go' '(prompt|completion|refusal|arguments|provider_body|authorization)' internal/aigateway/telemetry.go; then
  echo "T3 telemetry names raw model or credential material" >&2
  exit 1
fi

gofmt_files="$(gofmt -l $(find cmd/ai-gateway internal/aigateway -name '*.go' -type f))"
if [[ -n "$gofmt_files" ]]; then
  echo "T3 gateway files require gofmt:" >&2
  echo "$gofmt_files" >&2
  exit 1
fi

go test -race ./internal/aigateway ./cmd/ai-gateway
go test -run '^TestGatewayOpenAPIParity$' ./internal/aigateway
go test -run '^$' -bench 'Benchmark(WarmRouting|AuthenticatedHTTP)$' -benchtime=1x ./internal/aigateway

echo "T3 AI gateway transport acceptance passed."
