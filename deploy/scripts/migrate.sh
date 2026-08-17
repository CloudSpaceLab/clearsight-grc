#!/usr/bin/env bash
set -Eeuo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"
migrations_dir="${1:-migrations}"
test -d "$migrations_dir"

psql -X "$DATABASE_URL" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
CREATE TABLE IF NOT EXISTS public.clearsight_schema_migrations (
    filename text PRIMARY KEY,
    checksum_sha256 text NOT NULL CHECK (checksum_sha256 ~ '^[0-9a-f]{64}$'),
    applied_at timestamptz NOT NULL DEFAULT now()
);
SQL

while IFS= read -r migration; do
  filename="$(basename "$migration")"
  if [[ ! "$filename" =~ ^[0-9]{6}_[a-z0-9_]+\.up\.sql$ ]]; then
    echo "invalid migration filename: $filename" >&2
    exit 1
  fi

  checksum="$(sha256sum "$migration" | awk '{print $1}')"
  recorded="$(psql -XAt "$DATABASE_URL" -v ON_ERROR_STOP=1 -v filename="$filename" <<'SQL'
SELECT checksum_sha256
FROM public.clearsight_schema_migrations
WHERE filename = :'filename';
SQL
)"

  if [[ -n "$recorded" ]]; then
    if [[ "$recorded" != "$checksum" ]]; then
      echo "checksum mismatch: $filename" >&2
      exit 1
    fi
    continue
  fi

  first_statement="$(awk 'NF { print; exit }' "$migration" | tr -d '\r')"
  last_statement="$(awk 'NF { statement=$0 } END { print statement }' "$migration" | tr -d '\r')"
  transaction_markers="$(grep -Ec '^[[:space:]]*(BEGIN|COMMIT);[[:space:]]*$' "$migration" || true)"
  if [[ "$first_statement" != "BEGIN;" || "$last_statement" != "COMMIT;" || "$transaction_markers" != "2" ]]; then
    echo "migration must have one outer BEGIN/COMMIT pair: $filename" >&2
    exit 1
  fi

  transaction_file="$(mktemp)"
  trap 'rm -f "${transaction_file:-}"' EXIT
  awk '!/^[[:space:]]*(BEGIN|COMMIT);[[:space:]]*$/' "$migration" > "$transaction_file"
  cat >> "$transaction_file" <<'SQL'
INSERT INTO public.clearsight_schema_migrations(filename, checksum_sha256)
VALUES (:'filename', :'checksum');
SQL

  echo "applying $filename"
  psql -X -1 "$DATABASE_URL" -v ON_ERROR_STOP=1 -v filename="$filename" -v checksum="$checksum" \
    -f "$transaction_file" >/dev/null
  rm -f "$transaction_file"
  transaction_file=""
done < <(find "$migrations_dir" -maxdepth 1 -type f -name '*.up.sql' -print | LC_ALL=C sort)
