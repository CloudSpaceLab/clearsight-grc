#!/usr/bin/env bash
set -Eeuo pipefail

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
timeout 15 openssl s_client -connect "$CLEARSIGHT_SMTP_HOST:$CLEARSIGHT_SMTP_PORT" -servername "$CLEARSIGHT_SMTP_HOST" -starttls smtp </dev/null 2>/dev/null | openssl x509 -noout -checkend 0 >/dev/null 2>&1

printf 'email readiness verified\n'
