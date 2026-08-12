#!/usr/bin/env bash
set -Eeuo pipefail

domain="${1:?domain is required}"
database_password="${2:?database password is required}"
ci_public_key="${3:?CI public key is required}"
root=/opt/clearsight-grc
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
deploy_dir="$(cd "$script_dir/.." && pwd)"

[[ "$domain" == clearsight.cloudspacetechs.com ]]
[[ "$database_password" =~ ^[0-9a-f]{64}$ ]]
[[ "$ci_public_key" == ssh-ed25519\ * ]]
# shellcheck disable=SC1091
source /etc/os-release
[[ "$ID" == rocky ]]
systemctl is-active --quiet docker
systemctl is-active --quiet nginx
systemctl is-active --quiet postgresql-18
sudo -u postgres psql -XAtqc 'select 1' | grep -qx 1

install -d -m 0750 "$root" "$root/config" "$root/data" "$root/data/artifacts" \
  "$root/incoming" "$root/releases" "$root/state"

sudo -u postgres psql -X -v ON_ERROR_STOP=1 -v role_password="$database_password" <<'SQL'
SELECT format('CREATE ROLE clearsight LOGIN PASSWORD %L', :'role_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clearsight') \gexec
ALTER ROLE clearsight LOGIN PASSWORD :'role_password';
SQL

sudo -u postgres psql -X -v ON_ERROR_STOP=1 <<'SQL'
SELECT 'CREATE DATABASE clearsight OWNER clearsight'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'clearsight') \gexec
SQL
owner="$(sudo -u postgres psql -XAtqc "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname='clearsight'")"
[[ "$owner" == clearsight ]]

hba_file="$(sudo -u postgres psql -XAtqc 'show hba_file')"
hba_rule='host clearsight clearsight 127.0.0.1/32 scram-sha-256'
if ! grep -Fqx "$hba_rule" "$hba_file"; then
  hba_temp="$(mktemp)"
  {
    printf '%s\n' '# ClearSight demo application (managed by bootstrap-server.sh)'
    printf '%s\n' "$hba_rule"
    cat "$hba_file"
  } > "$hba_temp"
  install -o postgres -g postgres -m 0600 "$hba_temp" "$hba_file"
  rm -f "$hba_temp"
fi
sudo -u postgres psql -XAtqc 'select pg_reload_conf()' | grep -qx t

umask 077
cat > "$root/config/app.env" <<EOF
CLEARSIGHT_ENV=development
CLEARSIGHT_DEMO_MODE=true
CLEARSIGHT_IDENTITY_MODE=development
CLEARSIGHT_COMMAND_AUTHORIZATION=audit
CLEARSIGHT_ALLOWED_ORIGIN=https://$domain
CLEARSIGHT_ARTIFACT_ROOT=/var/lib/clearsight/artifacts
CLEARSIGHT_DOCUMENT_IMPORT_ALLOW_UNSCANNED_ANALYSIS=true
CLEARSIGHT_LOG_LEVEL=info
DATABASE_URL=postgres://clearsight:$database_password@127.0.0.1:5432/clearsight?sslmode=disable
EOF
chmod 0600 "$root/config/app.env"

install -m 0755 "$script_dir/ci-entrypoint.sh" /usr/local/sbin/clearsight-ci-entrypoint
install -d -m 0700 /root/.ssh
touch /root/.ssh/authorized_keys
chmod 0600 /root/.ssh/authorized_keys
key_line="command=\"/usr/local/sbin/clearsight-ci-entrypoint\",no-agent-forwarding,no-port-forwarding,no-pty,no-user-rc,no-X11-forwarding $ci_public_key clearsight-ci"
grep -Fqx "$key_line" /root/.ssh/authorized_keys || printf '%s\n' "$key_line" >> /root/.ssh/authorized_keys

install -d -m 0755 /var/www/certbot
install -m 0644 "$deploy_dir/nginx/clearsight-http.conf" /etc/nginx/conf.d/clearsight.conf
nginx -t
systemctl reload nginx
certbot certonly --webroot -w /var/www/certbot -d "$domain" \
  --non-interactive --agree-tos --register-unsafely-without-email --keep-until-expiring
install -m 0644 "$deploy_dir/nginx/clearsight.conf" /etc/nginx/conf.d/clearsight.conf
nginx -t
systemctl reload nginx
echo "ClearSight bootstrap complete"
