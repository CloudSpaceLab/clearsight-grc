#!/usr/bin/env bash
set -Eeuo pipefail

expected_sha="${1:?expected sha is required}"
[[ "$expected_sha" =~ ^[0-9a-f]{40}$ ]]

required=(
  CLEARSIGHT_RECIPIENT_KEYRING
  CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID
  CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY
  CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL
  CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED
  CLEARSIGHT_SMTP_HOST
  CLEARSIGHT_SMTP_PORT
  CLEARSIGHT_SMTP_USERNAME
  CLEARSIGHT_SMTP_PASSWORD
  CLEARSIGHT_SMTP_FROM
  CLEARSIGHT_SMTP_TLS_MODE
)
for name in "${required[@]}"; do
  [[ -n "${!name:-}" ]] || { printf 'email readiness missing required configuration name: %s\n' "$name" >&2; exit 1; }
done

[[ "$CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED" == "true" ]] || { printf 'email readiness requires external distribution delivery to be enabled\n' >&2; exit 1; }
[[ "$CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL" == https://* ]] || { printf 'email readiness requires an HTTPS public capture base URL\n' >&2; exit 1; }
[[ "$CLEARSIGHT_SMTP_TLS_MODE" == "STARTTLS" ]] || { printf 'email readiness requires STARTTLS\n' >&2; exit 1; }
[[ "$CLEARSIGHT_SMTP_PORT" =~ ^[0-9]{1,5}$ ]] && (( CLEARSIGHT_SMTP_PORT > 0 && CLEARSIGHT_SMTP_PORT <= 65535 )) || { printf 'email readiness requires a valid SMTP port\n' >&2; exit 1; }
[[ "$CLEARSIGHT_SMTP_FROM" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]] || { printf 'email readiness requires a valid sender address\n' >&2; exit 1; }
[[ "${CLEARSIGHT_SMTP_SECRET_REF:-}" == "env:CLEARSIGHT_SMTP_PASSWORD" ]] || { printf 'email readiness requires the protected SMTP password reference\n' >&2; exit 1; }

python3 - <<'PY' >/dev/null 2>&1
import base64, json, os
keys = json.loads(os.environ["CLEARSIGHT_RECIPIENT_KEYRING"])
active = os.environ["CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID"]
assert isinstance(keys, dict) and 1 <= len(keys) <= 8 and active in keys
for value in keys.values():
    assert len(base64.b64decode(value, validate=True)) == 32
assert len(base64.b64decode(os.environ["CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY"], validate=True)) == 32
PY

timeout 10 bash -c 'exec 3<>"/dev/tcp/$1/$2"' _ "$CLEARSIGHT_SMTP_HOST" "$CLEARSIGHT_SMTP_PORT" >/dev/null 2>&1
timeout 15 openssl s_client -connect "$CLEARSIGHT_SMTP_HOST:$CLEARSIGHT_SMTP_PORT" -servername "$CLEARSIGHT_SMTP_HOST" -verify_hostname "$CLEARSIGHT_SMTP_HOST" -verify_return_error -starttls smtp </dev/null >/dev/null 2>&1

ready="$(curl --fail --silent --show-error http://127.0.0.1:13281/health/ready)"
python3 -c 'import json,sys; value=json.load(sys.stdin); assert value == {"mode":"postgres","revision":sys.argv[1],"status":"ready"}' "$expected_sha" <<<"$ready"

worker_id="$(docker ps -q --filter label=com.cloudspacelab.clearsight=true --filter ancestor="clearsight-worker:$expected_sha")"
[[ -n "$worker_id" ]]
[[ "$(docker inspect -f '{{.State.Status}}' "$worker_id")" == "running" ]]
[[ "$(docker inspect -f '{{ index .Config.Labels \"org.opencontainers.image.revision\" }}' "$worker_id")" == "$expected_sha" ]]

printf '%s\n' \
  'smtp_configured=true' \
  'starttls_required=true' \
  'recipient_protection_configured=true' \
  'capture_origin_secure=true' \
  'api_revision_matches=true' \
  'worker_revision_matches=true'
