#!/usr/bin/env bash
set -euo pipefail

# V1 browser clients must consume the one documented, lowercase response contract.
if grep -En "raw\.[A-Z][A-Za-z]*|raw\[['\"][A-Z][A-Za-z]*" web/src/formsDistributionApi.ts; then
  echo "alternate response-key normalization remains in formsDistributionApi.ts" >&2
  exit 1
else
  scan_status=$?
  # Only a successful scan with no matches may proceed.
  if [ "$scan_status" -ne 1 ]; then
    exit "$scan_status"
  fi
fi

go test ./internal/httpapi -run 'Test(FormDistribution|GovernedFormDistribution|ResponseRevisionJSON|RuntimeOpenAPI)' -count=1
(
  cd web
  node node_modules/vitest/vitest.mjs run src/formsDistributionApi.test.ts --maxWorkers=1
)
