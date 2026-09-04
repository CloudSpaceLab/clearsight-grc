#!/usr/bin/env bash
set -euo pipefail

# V1 browser clients must consume the one documented, lowercase response contract.
if rg -n 'raw\.(ID|Status|Version|UpdatedAt|FormTemplateID|DistributionID)|raw\[["'"'][A-Z][A-Za-z]+' web/src/formsDistributionApi.ts; then
  echo "alternate response-key normalization remains in formsDistributionApi.ts" >&2
  exit 1
fi

go test ./internal/httpapi -run 'Test(FormDistributionResponseDTOs|RuntimeOpenAPI)' -count=1
(
  cd web
  node node_modules/vitest/vitest.mjs run src/formsDistributionApi.test.ts --maxWorkers=1
)
