#!/usr/bin/env bash
set -Eeuo pipefail

sha="${1:?sha is required}"
stage="${2:?stage is required}"
root=/opt/clearsight-grc
config="$root/config/app.env"
release="$root/releases/$sha"
compose="$release/compose.demo.yaml"
lock=/run/lock/clearsight-deploy.lock
phase=prepare
previous=""

[[ "$sha" =~ ^[0-9a-f]{40}$ ]]
[[ "$stage" == "$root/incoming/$sha."* ]]
exec 9>"$lock"
flock -n 9
(( $(df --output=avail -B1 "$root" | tail -1) >= 5368709120 )) || {
  echo "less than 5 GiB available" >&2
  exit 1
}
test -f "$config"
test ! -e "$release"

docker load -i "$stage/images.tar" >/dev/null
for component in api worker web; do
  image="clearsight-$component:$sha"
  test "$(docker image inspect -f '{{ index .Config.Labels "com.cloudspacelab.clearsight" }}' "$image")" = true
  test "$(docker image inspect -f '{{ index .Config.Labels "org.opencontainers.image.revision" }}' "$image")" = "$sha"
done

install -d -m 0750 "$release" "$release/scripts" "$release/migrations"
install -m 0640 "$stage/compose.demo.yaml" "$compose"
install -m 0700 "$stage/scripts/migrate.sh" "$release/scripts/migrate.sh"
install -m 0700 "$stage/scripts/seed-demo-foundation.sh" "$release/scripts/seed-demo-foundation.sh"
install -m 0700 "$stage/scripts/verify-hosted-release.sh" "$release/scripts/verify-hosted-release.sh"
install -m 0700 "$stage/scripts/verify-email-readiness.sh" "$release/scripts/verify-email-readiness.sh"
cp -a "$stage/migrations/." "$release/migrations/"
if [[ -f "$stage/schema-backward-compatible" ]]; then
  install -m 0640 "$stage/schema-backward-compatible" "$release/schema-backward-compatible"
fi

set -a
# shellcheck disable=SC1090
source "$config"
set +a
export CLEARSIGHT_IMAGE_TAG="$sha"
previous="$(cat "$root/state/current-sha" 2>/dev/null || true)"

rollback() {
  status=$?
  if [[ "$phase" == running && -n "$previous" && -f "$release/schema-backward-compatible" ]] &&
     grep -qx true "$release/schema-backward-compatible"; then
    export CLEARSIGHT_IMAGE_TAG="$previous"
    docker compose -p clearsight --env-file "$config" -f "$root/releases/$previous/compose.demo.yaml" up -d --no-build --remove-orphans || true
  elif [[ "$phase" == running ]]; then
    docker compose -p clearsight --env-file "$config" -f "$compose" stop api worker web || true
    echo "post-migration failure requires operator intervention" >&2
  fi
  exit "$status"
}
trap rollback ERR

"$release/scripts/migrate.sh" "$release/migrations"
"$release/scripts/seed-demo-foundation.sh"
docker run --rm --network host --env-file "$config" \
  --entrypoint /clearsight-seed-bank-reference "clearsight-api:$sha" \
  -tenant 00000000-0000-4000-8000-000000000001 \
  -legal-entity 00000000-0000-4000-8000-000000000002 \
  -actor 00000000-0000-4000-8000-000000000101 \
  -owner 00000000-0000-4000-8000-000000000107 \
  -contributor 00000000-0000-4000-8000-000000000108 \
  -reviewer 00000000-0000-4000-8000-000000000106 \
  -signatory 00000000-0000-4000-8000-000000000102 >/dev/null
phase=running
docker compose -p clearsight --env-file "$config" -f "$compose" up -d --no-build --remove-orphans

for attempt in $(seq 1 60); do
  api_id="$(docker compose -p clearsight --env-file "$config" -f "$compose" ps -q api)"
  api_health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$api_id" 2>/dev/null || true)"
  if curl -fsS http://127.0.0.1:13281/health/ready >/dev/null 2>&1 &&
     curl -fsS http://127.0.0.1:13280/healthz >/dev/null 2>&1 &&
     [[ "$api_health" == healthy ]]; then
    break
  fi
  [[ "$attempt" -lt 60 ]]
  sleep 2
done
"$release/scripts/verify-hosted-release.sh" "$sha"

if [[ -n "$previous" && "$previous" != "$sha" ]]; then
  printf '%s\n' "$previous" > "$root/state/previous-sha.new"
  mv "$root/state/previous-sha.new" "$root/state/previous-sha"
fi
printf '%s\n' "$sha" > "$root/state/current-sha.new"
mv "$root/state/current-sha.new" "$root/state/current-sha"
phase=healthy
trap - ERR

while IFS= read -r image_id; do
  tags="$(docker image inspect -f '{{join .RepoTags " "}}' "$image_id")"
  [[ "$tags" == *"clearsight-"* ]] || continue
  if [[ "$tags" == *":$sha"* ]] || [[ -n "$previous" && "$tags" == *":$previous"* ]]; then
    continue
  fi
  docker image rm "$image_id" >/dev/null 2>&1 || true
done < <(docker image ls -q --filter label=com.cloudspacelab.clearsight=true | sort -u)

echo "deployed $sha"
