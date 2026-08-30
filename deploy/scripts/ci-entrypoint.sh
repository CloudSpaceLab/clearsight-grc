#!/usr/bin/env bash
set -Eeuo pipefail

readonly root=/opt/clearsight-grc
readonly command_pattern='^deploy ([0-9a-f]{40})$'
if [[ ! "${SSH_ORIGINAL_COMMAND:-}" =~ $command_pattern ]]; then
  echo "unsupported deployment command" >&2
  exit 64
fi
sha="${BASH_REMATCH[1]}"

umask 077
mkdir -p "$root/incoming"
stage="$(mktemp -d "$root/incoming/${sha}.XXXXXX")"
trap 'rm -rf -- "$stage"' EXIT

archive="$stage/release.tar"
dd bs=1M count=4097 iflag=fullblock status=none of="$archive"
if (( $(stat -c %s "$archive") > 4294967296 )); then
  echo "release bundle exceeds 4 GiB" >&2
  exit 65
fi

while IFS= read -r member; do
  if [[ "$member" == /* || "$member" == ../* || "$member" == */../* || "$member" == *'/..' ]]; then
    echo "unsafe release path: $member" >&2
    exit 65
  fi
done < <(tar -tf "$archive")
if tar -tvf "$archive" | awk '$1 ~ /^[lh]/ { bad=1 } END { exit bad ? 0 : 1 }'; then
  echo "release bundle may not contain links" >&2
  exit 65
fi

tar --no-same-owner --no-same-permissions -xf "$archive" -C "$stage"
test -f "$stage/images.tar"
test -f "$stage/compose.demo.yaml"
test -d "$stage/migrations"
test -f "$stage/scripts/migrate.sh"
test -f "$stage/scripts/seed-demo-foundation.sh"
test -f "$stage/scripts/release.sh"
test -f "$stage/scripts/verify-hosted-release.sh"
chmod 0700 "$stage/scripts/migrate.sh" "$stage/scripts/seed-demo-foundation.sh" "$stage/scripts/release.sh" "$stage/scripts/verify-hosted-release.sh"
"$stage/scripts/release.sh" "$sha" "$stage"
