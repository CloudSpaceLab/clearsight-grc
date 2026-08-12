# Demo deployment

`clearsight.cloudspacetechs.com` is a persistent non-production demonstration environment. The normal URL presents the guided demo. Adding the exact query string `?demo=0` changes only the browser presentation to a reduced “Live preview · Non-production” view; it does not enable production configuration, bypass authentication, change authorization, or select a different database.

## Runtime boundary

GitHub Actions deploys only a successful, current `main` revision. The workflow builds immutable `clearsight-api`, `clearsight-worker`, and `clearsight-web` images tagged with the tested commit SHA and sends them through a dedicated forced-command SSH key. Host Nginx terminates TLS and proxies only to loopback ports `13280` (web) and `13281` (API).

The application reuses the server's native PostgreSQL 18 service. It owns only the `clearsight` role and database; Compose intentionally contains no PostgreSQL service or `5432` publication. Application state lives under `/opt/clearsight-grc`, with the environment file at `config/app.env`, artifacts at `data/artifacts`, immutable release inputs at `releases/<sha>`, and deployment state at `state/current-sha` and `state/previous-sha`.

This environment deliberately uses development identity, demo sessions, audit-mode command authorization, local artifact storage, and unscanned local document analysis. It is not production-ready; the production boundaries listed in the repository README still apply.

## GitHub Actions secrets

- `CLEARSIGHT_DEPLOY_KEY`: the dedicated Ed25519 private key whose public half is forced to `/usr/local/sbin/clearsight-ci-entrypoint` on the server.
- `CLEARSIGHT_DEPLOY_KNOWN_HOSTS`: pinned `ssh-keyscan` lines for `139.162.40.237`, verified against the server's local host-key fingerprints.
- `CLEARSIGHT_DEPLOY_HOST`: `139.162.40.237`.
- `CLEARSIGHT_DEPLOY_USER`: `root`.

Do not use the general-purpose server PEM as the Actions deploy key.

## Bootstrap

Run `deploy/scripts/bootstrap-server.sh` as root with the domain, a generated 64-character hexadecimal database password, and the dedicated Ed25519 public key. It is idempotent: it validates Rocky Linux and the active Docker, Nginx, and PostgreSQL 18 services; creates only ClearSight paths and database objects; adds one exact loopback SCRAM rule; installs the forced-command receiver; and obtains or renews the site's Let's Encrypt certificate.

The bootstrap never restarts PostgreSQL, prunes Docker globally, or modifies another domain's Nginx file. Validate afterward with:

```bash
systemctl is-active postgresql-18 nginx docker
sudo -u postgres psql -XAtqc "select datname, pg_get_userbyid(datdba) from pg_database where datname='clearsight'"
nginx -t
curl -fsS https://clearsight.cloudspacetechs.com/health/ready
docker compose -p clearsight --env-file /opt/clearsight-grc/config/app.env \
  -f "/opt/clearsight-grc/releases/$(cat /opt/clearsight-grc/state/current-sha)/compose.demo.yaml" ps
```

## Migrations and rollback

Only `*.up.sql` migrations run during deployment. The runner validates the repository's single outer `BEGIN/COMMIT` convention, then applies each migration and its SHA-256 ledger record atomically. Repeating a deployment is a no-op; changing an already-recorded migration fails before application startup.

A failure before migrations leaves the current containers untouched. After a migration starts, automatic rollback occurs only when the release explicitly declares schema backward compatibility. Current releases declare `false`, so a failed post-migration start stops the new ClearSight containers and requires an operator to diagnose the logs and either deploy a forward fix or explicitly restore a compatible prior release. Never use down migrations automatically.

Inspect state and logs with:

```bash
cat /opt/clearsight-grc/state/current-sha
cat /opt/clearsight-grc/state/previous-sha 2>/dev/null || true
docker compose -p clearsight --env-file /opt/clearsight-grc/config/app.env \
  -f "/opt/clearsight-grc/releases/$(cat /opt/clearsight-grc/state/current-sha)/compose.demo.yaml" logs --tail=200
sudo -u postgres psql -Xd clearsight -c 'table public.clearsight_schema_migrations order by filename'
```

Certificate renewal remains managed by the host's existing Certbot installation. After any renewal or Nginx change, run `nginx -t` before `systemctl reload nginx`.
