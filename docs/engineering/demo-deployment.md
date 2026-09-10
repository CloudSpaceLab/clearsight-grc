# Demo deployment

`clearsight.cloudspacetechs.com` is a persistent non-production demonstration environment. The normal URL presents the guided demo. Adding the exact query string `?demo=0` changes only the browser presentation to a reduced “Live preview · Non-production” view; it does not enable production configuration, bypass authentication, change authorization, or select a different database.

## Runtime boundary

GitHub Actions deploys only a successful, current `main` revision. The workflow builds immutable `clearsight-api`, `clearsight-worker`, and `clearsight-web` images tagged with the tested commit SHA and sends them through a dedicated forced-command SSH key. Host Nginx terminates TLS and proxies only to loopback ports `13280` (web) and `13281` (API).

The application reuses the server's native PostgreSQL 18 service. It owns only the `clearsight` role and database; Compose intentionally contains no PostgreSQL service or `5432` publication. Application state lives under `/opt/clearsight-grc`, with the environment file at `config/app.env`, artifacts at `data/artifacts`, immutable release inputs at `releases/<sha>`, and deployment state at `state/current-sha` and `state/previous-sha`.

This environment deliberately uses development identity, demo sessions, audit-mode command authorization, local artifact storage, and unscanned local document analysis. It is not production-ready; the production boundaries listed in the repository README still apply.

PostgreSQL lookups may accept either the demo tenant slug or UUID, but actor-facing continuity, evidence and workflow records return the canonical tenant UUID. Deployment also runs the idempotent demo foundation fixture: it maintains one active `CLEARSIGHT-DEMO-AUTHORITY` policy, records GRC Administrator as maker and Internal Auditor as the independent checker, and projects direct routes for the material workflow responsibilities. Incompatible stable-ID or policy-code collisions fail the deployment instead of overwriting unrelated governance data.

## Demo seed mode

Set `CLEARSIGHT_DEMO_SEED_MODE=manual` in the protected `/opt/clearsight-grc/config/app.env` to manage demonstration records separately from releases. In manual mode, releases skip the broad bank-reference seeder, including its operating vendor, history and acceptance fixtures. Migrations and the identity/authority foundation still run; image ownership/revision checks, application health, hosted verification and release promotion retain their existing gates. This mode does not delete or reconcile existing records and does not certify that the remaining data matches an external source.

The default when unset or empty is `reference`; explicitly setting `reference` also preserves automatic bank-reference seeding on each release. Any other value fails before images are loaded, release directories are installed or database changes begin. The setting persists in host configuration across releases; switching back to `reference` can recreate fixtures removed during manual curation. Keep manual mode enabled while maintaining a source-curated demonstration.

The hosted verifier checks authenticated response structure, not a minimum fixture count, so manual mode requires no relaxed data or security assertions. `deploy/tests/release_seed_mode_test.py` executes both modes with isolated command adapters and proves invalid configuration and failed required checks cannot promote a release.

## Application readiness and SMTP availability

Deployment requires the expected API revision, PostgreSQL readiness, web health, the owned worker running at the expected revision, authenticated demo reads and safe denial of invalid form access. These checks remain blocking. When `VERIFY_EMAIL_READINESS=true`, required email configuration, recipient encryption/HMAC keys, HTTPS capture origin and STARTTLS configuration also remain blocking.

Only the external SMTP TCP and certificate-validated STARTTLS probes are advisory during hosted deployment verification. A failed probe prints a redacted warning and `smtp_connectivity=unavailable`; it does not stop containers, roll back or prevent release-state promotion after the required checks pass. Probe time is bounded to 10 seconds for TCP and 15 seconds for STARTTLS, with STARTTLS skipped if TCP fails. A successful probe reports `smtp_connectivity=available`, not successful authentication, delivery or inbox receipt.

For explicit email acceptance, run `deploy/scripts/verify-email-readiness.sh <sha>` with the protected configuration loaded and shell tracing off. Its default remains strict. The hosted verifier supplies `--smtp-advisory` explicitly; that option cannot suppress configuration, security, API or worker failures. There is no catch-all ignored verification error and no change to runtime transport security.

An application deployment receipt is not an email acceptance receipt. Review actual delivery failures through existing Forms communications and system operations. Existing workers retain bounded retries for eligible failures; terminal and unknown post-acceptance outcomes require operator reconciliation and are not automatically replayed by deployment. Do not send test messages, disclose protected values or mark failed jobs resolved merely because a connectivity probe recovers.

This boundary follows the [approved SMTP advisory design](../superpowers/specs/2026-09-08-smtp-advisory-deployment-design.md). The executable regression suite is `python3 -m unittest discover -s deploy/tests -p '*test.py'` and runs in the existing backend CI job.

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
