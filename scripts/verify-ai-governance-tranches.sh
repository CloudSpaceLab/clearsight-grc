#!/usr/bin/env bash
set -Eeuo pipefail

mode="${1:-auto}"
required=(
  internal/aigateway/governance.go
  internal/aigovernance/model.go
  internal/aigovernance/service.go
  internal/aigovernance/runtime.go
  internal/aigovernance/retention.go
  internal/httpapi/ai_governance_handlers.go
  migrations/000035_ai_governance_enforcement.up.sql
  migrations/000035_ai_governance_enforcement.down.sql
  migrations/000036_ai_governance_receipts_grants.up.sql
  migrations/000036_ai_governance_receipts_grants.down.sql
  migrations/000076_ai_governance_baseline_attribution.up.sql
  migrations/000076_ai_governance_baseline_attribution.down.sql
  docs/acceptance/t4-governed-ai-enforcement.md
  docs/acceptance/t5-ai-governance-receipts-approval.md
)
for file in "${required[@]}"; do
  test -s "$file" || { echo "missing AI governance contract: $file" >&2; exit 1; }
done

ai_governance_migrations=(
  migrations/000035_ai_governance_enforcement.up.sql
  migrations/000036_ai_governance_receipts_grants.up.sql
  migrations/000076_ai_governance_baseline_attribution.up.sql
)
if grep -nE '(prompt|response_body|source_payload|provider_secret|authorization_header)' "${ai_governance_migrations[@]}"; then
  echo "AI governance durable schema contains prohibited raw-content or credential field names" >&2
  exit 1
fi
if ! grep -q "ai_gateway_decision_receipts" docs/architecture/durable-schema-ownership.md || \
   ! grep -q "ai_execution_grants" docs/architecture/durable-schema-ownership.md; then
  echo "AI governance durable tables are missing schema ownership classification" >&2
  exit 1
fi

gofmt_files="$(gofmt -l $(find internal/aigateway internal/aigovernance cmd/ai-gateway -name '*.go' -type f))"
if [[ -n "$gofmt_files" ]]; then
  echo "AI governance files require gofmt:" >&2
  echo "$gofmt_files" >&2
  exit 1
fi

go test -race ./internal/aigateway ./internal/aigovernance ./internal/httpapi
go test -tags postgres ./internal/aigovernance ./cmd/ai-gateway ./cmd/api ./cmd/worker

if [[ "$mode" == "postgres" || ( "$mode" == "auto" && -n "${TEST_DATABASE_URL:-}" ) ]]; then
  test -n "${TEST_DATABASE_URL:-}" || { echo "TEST_DATABASE_URL is required for PostgreSQL AI governance acceptance" >&2; exit 1; }
  go test -tags 'postgres postgresintegration' ./internal/aigovernance
fi

echo "T4/T5 AI governance acceptance passed, including organization-baseline attribution."
