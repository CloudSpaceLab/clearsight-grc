#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
expected_sha="${1:?expected sha is required}"
base_url="${2:-https://clearsight.cloudspacetechs.com}"
[[ "$expected_sha" =~ ^[0-9a-f]{40}$ ]]
if [[ "${VERIFY_EMAIL_READINESS:-false}" == "true" ]]; then
  "$script_dir/verify-email-readiness.sh" "$expected_sha"
fi

cookie_jar="$(mktemp)"
denial_body="$(mktemp)"
trap 'rm -f "$cookie_jar" "$denial_body"' EXIT
curl_retry=(--fail --silent --show-error --retry 6 --retry-delay 2 --retry-max-time 45 --retry-all-errors)

ready="$(curl "${curl_retry[@]}" "$base_url/health/ready")"
python3 -c 'import json,sys; value=json.load(sys.stdin); assert value == {"mode":"postgres","revision":sys.argv[1],"status":"ready"}' "$expected_sha" <<<"$ready"

curl "${curl_retry[@]}" -c "$cookie_jar" \
  -H 'Content-Type: application/json' \
  --data '{"username":"system-admin@demo.clearsight.local","password":"demo"}' \
  "$base_url/api/v1/demo/login" >/dev/null

session="$(curl "${curl_retry[@]}" -b "$cookie_jar" "$base_url/api/v1/session/status")"
python3 -c 'import json,sys; value=json.load(sys.stdin); assert value.get("authenticated") is True' <<<"$session"

for path in /api/v1/today '/api/v1/forms/templates?limit=1' '/api/v1/vendors?limit=1'; do
  body="$(curl "${curl_retry[@]}" -b "$cookie_jar" "$base_url$path")"
  python3 -c 'import json,sys; value=json.load(sys.stdin); assert isinstance(value,dict) and isinstance(value.get("items"),list)' <<<"$body"
done

denial_status=""
for _attempt in $(seq 1 6); do
  denial_status="$(curl -sS -o "$denial_body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    --data '{"route_selector":"invalid_access_selector"}' \
    "$base_url/api/v1/evidence/access/start")"
  if [[ "$denial_status" == 401 || "$denial_status" == 503 ]]; then
    break
  fi
  sleep 2
done
[[ "$denial_status" == 401 || "$denial_status" == 503 ]]
python3 -c 'import json,sys; value=json.load(open(sys.argv[1],encoding="utf-8")); expected={"401":"form_access_failed","503":"form_access_unavailable"}[sys.argv[2]]; assert value.get("error")==expected; forbidden={"distribution","recipient","audience_hint","route_selector","request_id"}; assert forbidden.isdisjoint(value)' "$denial_body" "$denial_status"

printf 'verified hosted release %s\n' "$expected_sha"
