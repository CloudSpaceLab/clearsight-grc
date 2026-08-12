# Main Demo Auto-Deployment Design

**Status:** Approved for planning

**Date:** 2026-08-12

**Target:** `https://clearsight.cloudspacetechs.com` on `139.162.40.237`

## Purpose

Deploy every accepted `main` revision of ClearSight to the existing shared server as an explicitly non-production stakeholder environment. The root URL presents the normal demo experience. Adding `?demo=0` presents a live-preview treatment backed by the same durable API and database, without claiming or enabling production security mode.

This deployment does not represent a production-ready banking environment. The repository's production identity, storage, malware-scanning, recovery and scale gates remain authoritative.

## Runtime architecture

The deployment consists of three ClearSight containers:

- a PostgreSQL-tagged Go API;
- a PostgreSQL-tagged durable worker;
- a production-built React application served by a small static Nginx container.

The host's existing Nginx remains the only public origin. It terminates TLS and routes `/api/`, `/health/` and any future documented machine edges such as `/scim/` to the ClearSight API on a dedicated loopback port. All other requests go to the static web container on a separate loopback port.

The API and worker use Linux host networking so they can connect to the existing native PostgreSQL service at `127.0.0.1:5432`. The API listens on a dedicated `127.0.0.1` port that does not collide with current services. The web container uses an explicit loopback-only port publication. No ClearSight application port is bound to a public interface.

## Existing PostgreSQL boundary

The deployment MUST NOT create or run a PostgreSQL container. It MUST NOT replace, restart, reconfigure, expose or prune the existing PostgreSQL service.

ClearSight receives:

- one dedicated database in the existing native cluster;
- one dedicated least-privilege login role;
- ownership or explicit privileges limited to the ClearSight database and schema;
- a generated password stored only in the root-owned server environment file and GitHub deployment secret where required.

Existing application databases, roles, schemas and connections remain out of scope. ClearSight migrations run only against the dedicated ClearSight database and stop on the first error. Initial provisioning is idempotent and refuses to repurpose a database owned by an unexpected role.

The initial schema is produced by applying all ordered `migrations/*.up.sql` files. Subsequent deployments record applied migration filenames and checksums in a deployment-owned migration ledger before applying only new migrations. A changed checksum for an already-applied migration fails the deployment.

## Demo and live-preview behavior

The backend always starts as a non-production deployment with development identity, demo mode enabled and PostgreSQL repositories enabled. Demo routes and supplied role credentials therefore remain explicitly non-production.

The default URL shows the current demo login, demo labels, supplied roles and reference journey affordances.

When the exact query parameter `demo=0` is present:

- the frontend marks the session as live preview;
- demo-only labels, role-switch controls, reference-journey launchers and sample-data fallback presentation are hidden;
- application reads and writes continue through the same live API and persistent ClearSight database;
- authentication, authorization and server capabilities are not changed;
- the frontend displays a concise non-production/live-preview notice where needed so the view cannot be mistaken for production.

Removing the query parameter returns to the default demo presentation. Query parameters never select backend environment, identity mode, command authorization policy, database or secrets. The server remains the security boundary.

## Build and release flow

The existing CI jobs remain the merge gate for `main`: Go formatting, race-enabled unit tests, PostgreSQL composition and integration tests, migration rollback/reapply, `go vet`, TypeScript checking, rendered-state/accessibility tests and the Vite production build.

After those jobs pass on a push to `main`, a deployment job:

1. checks out the exact commit SHA;
2. builds immutable API, worker and web images tagged with that SHA;
3. transfers the compressed images and a small release manifest to the server over SSH;
4. verifies image digests and available disk space;
5. loads the images without building source on the server;
6. applies pending ClearSight-only migrations to the existing database;
7. updates the ClearSight Compose project to the exact image tags;
8. waits for the API readiness endpoint and web response;
9. verifies the public domain and records the deployed SHA;
10. removes only superseded, unreferenced ClearSight image tags after a healthy release.

GitHub Actions concurrency allows only one production-target deployment at a time and cancels stale queued runs. The workflow receives `contents: read` only; it does not require a repository write token.

## Server layout and secrets

ClearSight deployment state lives under `/opt/clearsight-grc` with separate release, configuration and persistent artifact paths. Root owns the directory. Environment files are mode `0600` and never committed or printed.

The supplied root PEM is used only to bootstrap a dedicated CI deployment key. The GitHub workflow then connects with the dedicated key. Its server-side authorization is restricted as far as practical to the ClearSight deployment command. The broad root PEM is not committed or copied into the repository.

Required secrets include the SSH private key, host, user, deployment path and ClearSight database password. Workflow logs mask secret values and do not echo the environment file or database URL.

## Nginx and TLS

A dedicated `/etc/nginx/conf.d/clearsight.conf` owns only `clearsight.cloudspacetechs.com`. It follows the server's current TLS, forwarded-header and security-header conventions.

Before activation, the deployer validates the configuration with `nginx -t`. TLS uses the existing Certbot/Let's Encrypt convention, with Cloudflare proxying preserved. Nginx is reloaded only after successful validation. The configuration includes a bounded upload limit compatible with ClearSight's configured artifact limit and avoids caching API responses.

## Failure handling and rollback

Application images are immutable and the previous healthy SHA is retained. If image loading, container startup or health verification fails, the deployer restores the previous image tags and rechecks health.

Database migrations are forward-only during automatic deployment. The workflow does not automatically run destructive down migrations. If a new migration has committed successfully but the new application fails, automatic application rollback occurs only when the previous release is declared compatible with the new schema. Otherwise the deployment stops with the new containers disabled and reports that operator intervention is required.

The deployer never runs global Docker pruning. Cleanup is label- and project-scoped to unreferenced ClearSight images. PostgreSQL data, other Docker projects, other Nginx virtual hosts and unrelated `/opt` paths are never cleanup targets.

## Health, observability and acceptance

The API already exposes `/health/live` and `/health/ready`. Container health checks use the loopback API endpoint. Deployment acceptance requires:

- the API readiness endpoint returns HTTP 200 and reports PostgreSQL mode;
- the web root returns HTTP 200;
- the public HTTPS domain returns HTTP 200 through Cloudflare;
- `/api/v1/context` has the expected demo authentication behavior;
- the default URL offers demo login;
- `?demo=0` hides demo-only presentation and retains the non-production notice;
- the deployed SHA matches the triggering GitHub commit;
- existing server containers and native PostgreSQL remain running.

Deployment logs include the commit SHA, migration filenames, container health and rollback outcome, but never credentials, session material, tokens or protected data.

## Testing strategy

The query-driven presentation change is implemented test-first with focused React tests proving default demo and `demo=0` behavior. Deployment configuration receives static validation tests that assert no PostgreSQL service exists, all published ports are loopback-only, the images are SHA-pinned and the expected health checks/routes exist.

Local verification runs the complete existing release gates plus a production-image smoke test against an isolated test PostgreSQL database. Server verification is read-only until bootstrap and then is scoped to the new ClearSight database, containers, Nginx virtual host and deployment directory.

## Out of scope

- converting the environment into a production bank deployment;
- allowing a query string to change server security or identity mode;
- provisioning OIDC, SCIM or signed-gateway production credentials;
- replacing the local artifact adapter with production object storage;
- modifying or migrating another application's PostgreSQL database;
- global Docker cleanup or shared-server restructuring;
- changing Cloudflare account configuration unless the existing DNS/TLS path proves unusable.
