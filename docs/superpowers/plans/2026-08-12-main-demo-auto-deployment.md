# Main Demo Auto-Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically deploy each tested `main` commit to `clearsight.cloudspacetechs.com` as a persistent non-production demo that uses the server's existing PostgreSQL cluster and supports the safe `?demo=0` live-preview presentation.

**Architecture:** GitHub CI remains the release gate, then a separate successful-CI workflow builds immutable API, worker and web images and streams a release bundle over a forced-command SSH key. Host Nginx proxies loopback-only ClearSight services; the API and worker use host networking to reach a dedicated database in the existing native PostgreSQL cluster. The query parameter changes frontend presentation only and never changes backend environment, identity, authorization or database configuration.

**Tech Stack:** Go 1.26, React 19, TypeScript 7, Vite 8, Vitest, Docker/Compose, Nginx, PostgreSQL/psql, Bash, Python `unittest`, GitHub Actions, SSH.

---

## File map

- `web/src/runtimePresentation.ts`: parse the one presentation-only query switch.
- `web/src/runtimePresentation.test.ts`: prove exact query semantics.
- `web/src/App.tsx`: apply presentation mode to demo labels, reference navigation and fallback data.
- `web/src/App.test.tsx`: prove a demo backend is visually reduced in live preview.
- `web/src/components/DemoAuthGate.tsx`: keep demo authentication but suppress role switching in live preview.
- `web/src/components/DemoAuthGate.test.tsx`: prove authentication remains active while the switch control is hidden.
- `web/src/main.tsx`: compute presentation once and inject it into the application and auth gate.
- `cmd/healthcheck/main.go`: dependency-free readiness probe for the distroless API image.
- `cmd/healthcheck/main_test.go`: probe success/failure tests.
- `Dockerfile.api`: include the healthcheck binary and image health contract.
- `Dockerfile.web`: produce the immutable Vite/Nginx image.
- `.dockerignore`: keep build contexts bounded and secret-free.
- `deploy/web/nginx.conf`: serve the SPA and an internal web health endpoint.
- `deploy/compose.demo.yaml`: run API/worker/web only; never define PostgreSQL.
- `deploy/nginx/clearsight-http.conf`: ACME/bootstrap-only HTTP virtual host.
- `deploy/nginx/clearsight.conf`: final TLS reverse proxy.
- `deploy/scripts/migrate.sh`: checksum-ledger, forward-only migration runner.
- `deploy/scripts/release.sh`: load one immutable release, migrate, start, verify and narrowly roll back.
- `deploy/scripts/ci-entrypoint.sh`: stable forced-command receiver for CI release bundles.
- `deploy/scripts/bootstrap-server.sh`: idempotently create only ClearSight's database/role, paths, key restriction and Nginx host.
- `deploy/tests/deployment_config_test.py`: static safety assertions for Compose, Nginx, workflow and scripts.
- `.github/workflows/deploy-demo.yml`: build and transmit the exact CI-green `main` SHA.
- `docs/engineering/demo-deployment.md`: operator setup, secrets, rollback and non-production boundary.
- `README.md`: link the deployment guide without advertising production readiness.

### Task 1: Parse the presentation-only query switch

**Files:**
- Create: `web/src/runtimePresentation.test.ts`
- Create: `web/src/runtimePresentation.ts`

- [ ] **Step 1: Write the failing query contract tests**

```ts
import { describe, expect, it } from "vitest";
import { runtimePresentation } from "./runtimePresentation";

describe("runtimePresentation", () => {
  it.each(["", "?demo=1", "?demo=false", "?demo="])("keeps demo presentation for %s", (search) => {
    expect(runtimePresentation(search)).toBe("demo");
  });

  it("selects live preview only for the exact demo=0 value", () => {
    expect(runtimePresentation("?tour=off&demo=0")).toBe("live-preview");
  });
});
```

- [ ] **Step 2: Run the test and verify the module is missing**

Run: `cd web && npm test -- runtimePresentation.test.ts`

Expected: FAIL because `./runtimePresentation` cannot be resolved.

- [ ] **Step 3: Implement the minimal parser**

```ts
export type RuntimePresentation = "demo" | "live-preview";

export function runtimePresentation(search: string): RuntimePresentation {
  return new URLSearchParams(search).get("demo") === "0" ? "live-preview" : "demo";
}
```

- [ ] **Step 4: Run the focused test**

Run: `cd web && npm test -- runtimePresentation.test.ts`

Expected: PASS, 5 cases.

- [ ] **Step 5: Commit**

```bash
git add web/src/runtimePresentation.ts web/src/runtimePresentation.test.ts
git commit -m "feat(web): parse live preview query mode"
```

### Task 2: Apply live-preview presentation without weakening demo authentication

**Files:**
- Modify: `web/src/App.test.tsx`
- Modify: `web/src/App.tsx`
- Create: `web/src/components/DemoAuthGate.test.tsx`
- Modify: `web/src/components/DemoAuthGate.tsx`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Add a failing App test for a demo backend in live preview**

Add to `web/src/App.test.tsx`:

```tsx
it("uses live API data without demo-only presentation when requested", async () => {
  vi.mocked(loadContext).mockResolvedValue(runtime(true));
  render(<App presentation="live-preview" />);

  await screen.findByText("Live preview · Non-production");
  await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
  expect(screen.queryByText("Stakeholder demo")).toBeNull();
  expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
});
```

- [ ] **Step 2: Add a failing auth-gate test**

Create `web/src/components/DemoAuthGate.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadContext, loadDemoAccounts, logoutDemo } from "../api";
import { DemoAuthGate } from "./DemoAuthGate";

vi.mock("../api", () => ({
  loadContext: vi.fn(),
  loadDemoAccounts: vi.fn(),
  logoutDemo: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(loadContext).mockResolvedValue({
    tenant: { id: "bank-demo", name: "Demo Bank" },
    legal_entity: { id: "bank-ng", name: "Demo Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer" },
    mode: "postgres",
    demo_mode: true,
  });
  vi.mocked(loadDemoAccounts).mockResolvedValue([]);
  vi.mocked(logoutDemo).mockResolvedValue(undefined);
});

it("keeps demo authentication while hiding role switching in live preview", async () => {
  render(<DemoAuthGate presentation="live-preview"><div>Workspace</div></DemoAuthGate>);
  expect(await screen.findByText("Workspace")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Switch demo role" })).toBeNull();
});
```

- [ ] **Step 3: Run both tests and verify the missing props/behavior**

Run: `cd web && npm test -- App.test.tsx components/DemoAuthGate.test.tsx`

Expected: FAIL because `presentation` is not accepted and demo controls remain visible.

- [ ] **Step 4: Separate server demo capability from frontend presentation**

Change the App signature and derived mode:

```tsx
import type { RuntimePresentation } from "./runtimePresentation";

function App({ presentation = "demo" }: { presentation?: RuntimePresentation }) {
  // existing state
  const serverDemoMode = runtime?.demo_mode === true;
  const demoMode = serverDemoMode && presentation === "demo";
```

In the initial load effect, compute fallback with the prop and add it to the dependency list:

```ts
const allowFallback = (currentRuntime?.demo_mode === true && presentation === "demo") ||
  (currentRuntime == null && sampleMode && presentation === "demo");
```

Keep existing `demoMode` consumers so Explore, generic capture, the stakeholder label and `data-clearsight-demo` all turn off together. Add the alternate context label:

```tsx
{demoMode
  ? <mark>Stakeholder demo</mark>
  : serverDemoMode && presentation === "live-preview"
    ? <mark>Live preview · Non-production</mark>
    : null}
```

- [ ] **Step 5: Keep the demo login but hide role switching**

Change `DemoAuthGate` to accept the same optional prop and gate only the switch control:

```tsx
import type { RuntimePresentation } from "../runtimePresentation";

export function DemoAuthGate({ children, presentation = "demo" }: {
  children: ReactNode;
  presentation?: RuntimePresentation;
}) {
  // existing implementation
  const canSwitchRole = demoMode && presentation === "demo" && import.meta.env.VITE_STATIC_DEMO !== "true";
```

- [ ] **Step 6: Inject the parsed presentation once at startup**

In `web/src/main.tsx`:

```tsx
import { runtimePresentation } from "./runtimePresentation";

const presentation = runtimePresentation(window.location.search);
// preserve invitation and evidence-fixture precedence
const application = invitationToken
  ? <ExternalCaptureApp invitationToken={invitationToken}/>
  : lifecycleEvidence
    ? <LifecycleTodayEvidencePage/>
    : operatingEvidence
      ? <OperatingMutationsEvidencePage/>
      : <DemoAuthGate presentation={presentation}><App presentation={presentation}/></DemoAuthGate>;
```

- [ ] **Step 7: Run focused and full web verification**

Run: `cd web && npm test -- runtimePresentation.test.ts App.test.tsx components/DemoAuthGate.test.tsx && npm run typecheck && npm run build`

Expected: all tests PASS, TypeScript exits 0, Vite build exits 0.

- [ ] **Step 8: Commit**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/main.tsx web/src/components/DemoAuthGate.tsx web/src/components/DemoAuthGate.test.tsx
git commit -m "feat(web): add non-production live preview presentation"
```

### Task 3: Add an API image readiness probe

**Files:**
- Create: `cmd/healthcheck/main_test.go`
- Create: `cmd/healthcheck/main.go`
- Modify: `Dockerfile.api`

- [ ] **Step 1: Write failing probe behavior tests**

Create `cmd/healthcheck/main_test.go`:

```go
package main

import (
  "net/http"
  "net/http/httptest"
  "strings"
  "testing"
  "time"
)

func TestCheckAcceptsReadyResponse(t *testing.T) {
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
  defer server.Close()
  if err := check(&http.Client{Timeout: time.Second}, server.URL); err != nil { t.Fatalf("check: %v", err) }
}

func TestCheckRejectsUnreadyResponse(t *testing.T) {
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
  defer server.Close()
  err := check(&http.Client{Timeout: time.Second}, server.URL)
  if err == nil || !strings.Contains(err.Error(), "503") { t.Fatalf("expected 503 error, got %v", err) }
}

func TestCheckRejectsUnreachableTarget(t *testing.T) {
  if err := check(&http.Client{Timeout: 10 * time.Millisecond}, "http://127.0.0.1:1"); err == nil {
    t.Fatal("expected connection error")
  }
}
```

- [ ] **Step 2: Verify the healthcheck package is missing**

Run: `go test ./cmd/healthcheck`

Expected: FAIL because the package has no implementation.

- [ ] **Step 3: Implement the dependency-free probe**

```go
package main

import (
  "fmt"
  "net/http"
  "os"
  "time"
)

func check(client *http.Client, target string) error {
  response, err := client.Get(target)
  if err != nil { return err }
  defer response.Body.Close()
  if response.StatusCode != http.StatusOK { return fmt.Errorf("readiness returned %d", response.StatusCode) }
  return nil
}

func main() {
  target := "http://127.0.0.1:13281/health/ready"
  if len(os.Args) == 2 { target = os.Args[1] }
  if err := check(&http.Client{Timeout: 5 * time.Second}, target); err != nil {
    _, _ = fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}
```

- [ ] **Step 4: Verify the tests pass**

Run: `go test ./cmd/healthcheck`

Expected: PASS.

- [ ] **Step 5: Include the binary and health contract in `Dockerfile.api`**

Build `/out/clearsight-healthcheck`, copy it into the distroless image and add:

```dockerfile
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=12 \
  CMD ["/clearsight-healthcheck", "http://127.0.0.1:13281/health/ready"]
```

- [ ] **Step 6: Run Go verification and commit**

Run: `gofmt -w cmd/healthcheck && go test ./cmd/healthcheck && go test ./...`

```bash
git add cmd/healthcheck Dockerfile.api
git commit -m "feat(deploy): add API readiness probe"
```

### Task 4: Define immutable web and Compose runtime artifacts

**Files:**
- Create: `.dockerignore`
- Create: `Dockerfile.web`
- Create: `deploy/web/nginx.conf`
- Create: `deploy/compose.demo.yaml`
- Create: `deploy/tests/deployment_config_test.py`

- [ ] **Step 1: Write failing static safety tests**

Create the initial `deploy/tests/deployment_config_test.py`:

```python
from pathlib import Path
import unittest


class DeploymentConfigTest(unittest.TestCase):
    def read(self, relative: str) -> str:
        return Path(relative).read_text(encoding="utf-8")

    def test_compose_uses_existing_postgres_and_loopback_ports(self) -> None:
        compose = self.read("deploy/compose.demo.yaml")
        self.assertNotRegex(compose, r"(?m)^\s{2}postgres:")
        self.assertNotIn("5432:5432", compose)
        self.assertIn("network_mode: host", compose)
        self.assertIn('127.0.0.1:13280:80', compose)
        for component in ("api", "worker", "web"):
            self.assertIn(f"clearsight-{component}:${{CLEARSIGHT_IMAGE_TAG:?", compose)

    def test_docker_context_excludes_secrets_and_outputs(self) -> None:
        ignored = self.read(".dockerignore")
        for value in (".git", ".env.*", "web/node_modules", "web/dist", "var", "coverage"):
            self.assertIn(value, ignored)

    def test_web_image_serves_spa_and_health(self) -> None:
        nginx = self.read("deploy/web/nginx.conf")
        self.assertIn("try_files $uri $uri/ /index.html", nginx)
        self.assertIn("location = /healthz", nginx)


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run and verify failure because artifacts do not exist**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: FAIL with missing file errors.

- [ ] **Step 3: Create the bounded Docker context and web image**

Create `.dockerignore`:

```dockerignore
.git
.github
.env
.env.*
!.env.example
coverage
coverage.out
var
web/node_modules
web/dist
web/.vite
docs/screenshots
```

Create `Dockerfile.web`:

```dockerfile
FROM node:24.18.0-alpine AS build
WORKDIR /src
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
ENV VITE_API_BASE_URL=""
RUN npm run typecheck && npm test && npm run build

FROM nginx:1.30.4-alpine3.24
COPY deploy/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=6 \
  CMD wget -q -O - http://127.0.0.1/healthz >/dev/null || exit 1
```

Create `deploy/web/nginx.conf`:

```nginx
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    location = /healthz {
        access_log off;
        default_type text/plain;
        return 200 "ok\n";
    }

    location /assets/ {
        try_files $uri =404;
        add_header Cache-Control "public, max-age=31536000, immutable";
    }

    location / {
        try_files $uri $uri/ /index.html;
        add_header Cache-Control "no-store";
    }
}
```

- [ ] **Step 4: Create API/worker/web-only Compose**

Create `deploy/compose.demo.yaml`:

```yaml
services:
  api:
    image: clearsight-api:${CLEARSIGHT_IMAGE_TAG:?CLEARSIGHT_IMAGE_TAG is required}
    network_mode: host
    restart: unless-stopped
    env_file: /opt/clearsight-grc/config/app.env
    environment:
      CLEARSIGHT_HTTP_ADDR: 127.0.0.1:13281
    volumes:
      - /opt/clearsight-grc/data/artifacts:/var/lib/clearsight/artifacts
    labels:
      com.cloudspacelab.clearsight: "true"
      org.opencontainers.image.revision: ${CLEARSIGHT_IMAGE_TAG}

  worker:
    image: clearsight-worker:${CLEARSIGHT_IMAGE_TAG:?CLEARSIGHT_IMAGE_TAG is required}
    network_mode: host
    restart: unless-stopped
    env_file: /opt/clearsight-grc/config/app.env
    environment:
      CLEARSIGHT_WORKER_ID: worker-demo
    depends_on:
      api:
        condition: service_healthy
    labels:
      com.cloudspacelab.clearsight: "true"
      org.opencontainers.image.revision: ${CLEARSIGHT_IMAGE_TAG}

  web:
    image: clearsight-web:${CLEARSIGHT_IMAGE_TAG:?CLEARSIGHT_IMAGE_TAG is required}
    restart: unless-stopped
    ports:
      - "127.0.0.1:13280:80"
    depends_on:
      api:
        condition: service_healthy
    labels:
      com.cloudspacelab.clearsight: "true"
      org.opencontainers.image.revision: ${CLEARSIGHT_IMAGE_TAG}
```

- [ ] **Step 5: Run static tests**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add .dockerignore Dockerfile.web deploy/web/nginx.conf deploy/compose.demo.yaml deploy/tests/deployment_config_test.py
git commit -m "feat(deploy): define immutable demo runtime"
```

### Task 5: Implement checksum-ledger migrations for the existing PostgreSQL cluster

**Files:**
- Create: `deploy/scripts/migrate.sh`
- Modify: `.github/workflows/ci.yml`
- Modify: `deploy/tests/deployment_config_test.py`

- [ ] **Step 1: Extend static tests with migration safety assertions**

Add this method to `DeploymentConfigTest`:

```python
def test_migrations_are_forward_only_and_checksum_guarded(self) -> None:
    script = self.read("deploy/scripts/migrate.sh")
    for value in ("ON_ERROR_STOP", "clearsight_schema_migrations", "sha256sum", "checksum mismatch", "psql -X -1"):
        self.assertIn(value, script)
    self.assertIn("^[0-9]{6}_[a-z0-9_]+\\.up\\.sql$", script)
    self.assertNotIn(".down.sql", script)
```

- [ ] **Step 2: Run static tests and verify failure**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: FAIL because `deploy/scripts/migrate.sh` does not exist.

- [ ] **Step 3: Implement the forward-only runner**

Create `deploy/scripts/migrate.sh`:

```bash
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
  recorded="$(psql -XAt "$DATABASE_URL" -v ON_ERROR_STOP=1 -v filename="$filename" \
    -c "SELECT checksum_sha256 FROM public.clearsight_schema_migrations WHERE filename = :'filename'")"

  if [[ -n "$recorded" ]]; then
    if [[ "$recorded" != "$checksum" ]]; then
      echo "checksum mismatch: $filename" >&2
      exit 1
    fi
    continue
  fi

  echo "applying $filename"
  psql -X -1 "$DATABASE_URL" -v ON_ERROR_STOP=1 -v filename="$filename" -v checksum="$checksum" \
    -f "$migration" \
    -c "INSERT INTO public.clearsight_schema_migrations(filename, checksum_sha256) VALUES (:'filename', :'checksum')" \
    >/dev/null
done < <(find "$migrations_dir" -maxdepth 1 -type f -name '*.up.sql' -print | LC_ALL=C sort)
```

The success output is limited to migration filenames; it never prints `DATABASE_URL`.

- [ ] **Step 4: Add a CI integration proof on a fresh database**

After the existing migration gate in `ci.yml`, create `clearsight_deploy_test`, run the script twice, copy migrations to a temp directory, alter an already-applied file, and assert a third run fails with `checksum mismatch`. Drop only `clearsight_deploy_test` in an `if: always()` step.

- [ ] **Step 5: Run shell syntax, static and migration integration tests**

Run: `bash -n deploy/scripts/migrate.sh`

Run against the local/CI PostgreSQL fixture using the same commands added to `ci.yml`.

Expected: first apply PASS, second no-op PASS, altered checksum FAIL for the expected reason.

- [ ] **Step 6: Commit**

```bash
git add deploy/scripts/migrate.sh deploy/tests/deployment_config_test.py .github/workflows/ci.yml
git commit -m "feat(deploy): add forward-only migration ledger"
```

### Task 6: Implement constrained release reception, deployment and rollback

**Files:**
- Create: `deploy/scripts/ci-entrypoint.sh`
- Create: `deploy/scripts/release.sh`
- Modify: `deploy/tests/deployment_config_test.py`

- [ ] **Step 1: Add failing script-contract tests**

Add these methods:

```python
def test_forced_command_accepts_only_sha_deployments(self) -> None:
    script = self.read("deploy/scripts/ci-entrypoint.sh")
    self.assertIn("^deploy ([0-9a-f]{40})$", script)
    self.assertIn("/opt/clearsight-grc/incoming", script)
    self.assertIn("unsafe release path", script)
    self.assertIn('exec "$stage/scripts/release.sh"', script)

def test_release_is_scoped_and_health_checked(self) -> None:
    script = self.read("deploy/scripts/release.sh")
    for value in ("5368709120", "clearsight-api", "clearsight-worker", "clearsight-web",
                  "scripts/migrate.sh", "compose -p clearsight", "13281/health/ready",
                  "13280/healthz", "state/current-sha", "com.cloudspacelab.clearsight=true"):
        self.assertIn(value, script)
    for forbidden in ("docker system prune", "docker volume prune", "down -v", "systemctl restart postgresql"):
        self.assertNotIn(forbidden, script)
```

- [ ] **Step 2: Run tests and verify the missing scripts fail**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: FAIL with missing script errors.

- [ ] **Step 3: Implement the stable forced-command receiver**

Create `deploy/scripts/ci-entrypoint.sh`:

```bash
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
dd bs=1M count=4097 status=none of="$archive"
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
test -f "$stage/scripts/release.sh"
chmod 0700 "$stage/scripts/migrate.sh" "$stage/scripts/release.sh"
exec "$stage/scripts/release.sh" "$sha" "$stage"
```

- [ ] **Step 4: Implement the release state machine**

Create `deploy/scripts/release.sh` with the following exact state machine:

```bash
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
phase=running
docker compose -p clearsight --env-file "$config" -f "$compose" up -d --no-build --remove-orphans

for attempt in $(seq 1 60); do
  api_id="$(docker compose -p clearsight --env-file "$config" -f "$compose" ps -q api)"
  api_health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$api_id" 2>/dev/null || true)"
  curl -fsS http://127.0.0.1:13281/health/ready >/dev/null 2>&1 &&
    curl -fsS http://127.0.0.1:13280/healthz >/dev/null 2>&1 &&
    [[ "$api_health" == healthy ]] && break
  [[ "$attempt" -lt 60 ]]
  sleep 2
done
curl -fsS https://clearsight.cloudspacetechs.com/health/ready >/dev/null

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
  [[ "$tags" == *":$sha"* || (-n "$previous" && "$tags" == *":$previous"*) ]] && continue
  docker image rm "$image_id" >/dev/null 2>&1 || true
done < <(docker image ls -q --filter label=com.cloudspacelab.clearsight=true | sort -u)

echo "deployed $sha"
```

The implementation must:

1. lock `/run/lock/clearsight-deploy.lock` with `flock`;
2. require at least 5 GiB free;
3. load `images.tar` and inspect all three SHA tags/labels;
4. copy immutable release inputs to `/opt/clearsight-grc/releases/<sha>`;
5. run the migration script against the existing ClearSight database;
6. save the previous healthy SHA;
7. run `docker compose -p clearsight --env-file /opt/clearsight-grc/config/app.env -f ... up -d --no-build --remove-orphans`;
8. wait for API container health and curl loopback API/web endpoints;
9. curl the public HTTPS health endpoint;
10. atomically write `state/current-sha` and `state/previous-sha`;
11. on pre-migration failure, leave the current release untouched;
12. on post-start failure, restore the prior SHA only if `schema-backward-compatible=true` is present; otherwise stop the new app containers and report operator intervention;
13. remove only unreferenced ClearSight-labelled images older than the retained current/previous tags.

- [ ] **Step 5: Run syntax and static safety tests**

Run: `bash -n deploy/scripts/ci-entrypoint.sh deploy/scripts/release.sh deploy/scripts/migrate.sh`

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add deploy/scripts/ci-entrypoint.sh deploy/scripts/release.sh deploy/tests/deployment_config_test.py
git commit -m "feat(deploy): add constrained release state machine"
```

### Task 7: Add idempotent server bootstrap and Nginx ownership

**Files:**
- Create: `deploy/nginx/clearsight-http.conf`
- Create: `deploy/nginx/clearsight.conf`
- Create: `deploy/scripts/bootstrap-server.sh`
- Modify: `deploy/tests/deployment_config_test.py`

- [ ] **Step 1: Add failing bootstrap/Nginx safety tests**

Add these methods:

```python
def test_nginx_owns_only_the_clearsight_host(self) -> None:
    nginx = self.read("deploy/nginx/clearsight.conf")
    self.assertIn("server_name clearsight.cloudspacetechs.com", nginx)
    self.assertIn("127.0.0.1:13281", nginx)
    self.assertIn("127.0.0.1:13280", nginx)
    self.assertIn("/etc/letsencrypt/live/clearsight.cloudspacetechs.com", nginx)
    self.assertIn("proxy_no_cache 1", nginx)
    self.assertIn("client_max_body_size 21m", nginx)

def test_bootstrap_preserves_shared_services(self) -> None:
    script = self.read("deploy/scripts/bootstrap-server.sh")
    for value in ("CREATE ROLE clearsight", "CREATE DATABASE clearsight OWNER clearsight",
                  "/opt/clearsight-grc", "/etc/nginx/conf.d/clearsight.conf", "nginx -t"):
        self.assertIn(value, script)
    for forbidden in ("DROP DATABASE", "DROP ROLE", "systemctl restart postgresql", "docker system prune"):
        self.assertNotIn(forbidden, script)
```

- [ ] **Step 2: Run and verify failure**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: FAIL because bootstrap/Nginx artifacts are missing.

- [ ] **Step 3: Create HTTP bootstrap and final TLS virtual hosts**

Create `deploy/nginx/clearsight-http.conf`:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name clearsight.cloudspacetechs.com;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}
```

Create `deploy/nginx/clearsight.conf`:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name clearsight.cloudspacetechs.com;
    location /.well-known/acme-challenge/ { root /var/www/certbot; }
    location / { return 301 https://$host$request_uri; }
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name clearsight.cloudspacetechs.com;

    ssl_certificate /etc/letsencrypt/live/clearsight.cloudspacetechs.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/clearsight.cloudspacetechs.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    client_max_body_size 21m;

    location ~ ^/(api|health|scim)/ {
        proxy_pass http://127.0.0.1:13281;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_no_cache 1;
        proxy_cache_bypass 1;
        proxy_connect_timeout 10s;
        proxy_read_timeout 120s;
        proxy_send_timeout 120s;
    }

    location / {
        proxy_pass http://127.0.0.1:13280;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

- [ ] **Step 4: Implement idempotent bootstrap**

Create `deploy/scripts/bootstrap-server.sh`; it receives the domain, a generated hexadecimal database password and a dedicated CI public key:

```bash
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
systemctl is-active --quiet postgresql
sudo -u postgres psql -Atqc 'select 1' | grep -qx 1

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
certbot certonly --webroot -w /var/www/certbot -d "$domain" --non-interactive --agree-tos --keep-until-expiring
install -m 0644 "$deploy_dir/nginx/clearsight.conf" /etc/nginx/conf.d/clearsight.conf
nginx -t
systemctl reload nginx
echo "ClearSight bootstrap complete"
```

The implementation:

- verifies the host is the expected Rocky Linux server and Docker/Nginx/native PostgreSQL are active;
- creates `/opt/clearsight-grc/{config,data/artifacts,incoming,releases,state}` with root ownership and restrictive modes;
- creates role `clearsight` and database `clearsight` only when absent, and fails if an existing database has a different owner;
- writes `/opt/clearsight-grc/config/app.env` mode `0600` with development/demo identity, audit command authorization, the same-origin URL and `postgres://clearsight:<hex>@127.0.0.1:5432/clearsight?sslmode=disable`;
- installs the stable CI entrypoint mode `0755`;
- installs a forced-command authorized key with forwarding, PTY and agent forwarding disabled;
- installs the HTTP Nginx config, runs `nginx -t`, obtains/reuses the Let's Encrypt certificate, installs the TLS config, runs `nginx -t` again and reloads Nginx;
- never restarts PostgreSQL or another container.

- [ ] **Step 5: Run syntax/static tests and commit**

Run: `bash -n deploy/scripts/bootstrap-server.sh`

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

```bash
git add deploy/nginx deploy/scripts/bootstrap-server.sh deploy/tests/deployment_config_test.py
git commit -m "feat(deploy): add isolated server bootstrap"
```

### Task 8: Add the successful-CI deployment workflow

**Files:**
- Create: `.github/workflows/deploy-demo.yml`
- Modify: `deploy/tests/deployment_config_test.py`

- [ ] **Step 1: Add failing workflow safety tests**

Add this method:

```python
def test_workflow_deploys_only_green_current_main(self) -> None:
    workflow = self.read(".github/workflows/deploy-demo.yml")
    for value in ("workflow_run:", "workflows: [CI]", "conclusion == 'success'",
                  "workflow_run.event == 'push'", "head_branch == 'main'",
                  "workflow_run.head_sha", "cancel-in-progress: false", "contents: read",
                  "CLEARSIGHT_DEPLOY_KNOWN_HOSTS", '"deploy $RELEASE_SHA"'):
        self.assertIn(value, workflow)
    for component in ("api", "worker", "web"):
        self.assertIn(f'clearsight-{component}:$RELEASE_SHA', workflow)
    self.assertNotIn("bigbundle.pem", workflow)
```

- [ ] **Step 2: Run and verify the workflow test fails**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

Expected: FAIL because `deploy-demo.yml` is missing.

- [ ] **Step 3: Implement the workflow**

Core trigger and guard:

```yaml
name: Deploy demo
on:
  workflow_run:
    workflows: [CI]
    types: [completed]
permissions:
  contents: read
concurrency:
  group: clearsight-demo-deploy
  cancel-in-progress: false
jobs:
  deploy:
    if: >-
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.event == 'push' &&
      github.event.workflow_run.head_branch == 'main'
```

Create `.github/workflows/deploy-demo.yml`:

```yaml
name: Deploy demo
on:
  workflow_run:
    workflows: [CI]
    types: [completed]
permissions:
  contents: read
concurrency:
  group: clearsight-demo-deploy
  cancel-in-progress: false
jobs:
  deploy:
    if: >-
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.event == 'push' &&
      github.event.workflow_run.head_branch == 'main'
    runs-on: ubuntu-latest
    env:
      RELEASE_SHA: ${{ github.event.workflow_run.head_sha }}
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.event.workflow_run.head_sha }}
          fetch-depth: 0

      - name: Ignore superseded main revisions
        id: freshness
        run: |
          git fetch origin main
          if [ "$(git rev-parse origin/main)" = "$RELEASE_SHA" ]; then
            echo "current=true" >> "$GITHUB_OUTPUT"
          else
            echo "current=false" >> "$GITHUB_OUTPUT"
          fi

      - name: Build immutable images
        if: steps.freshness.outputs.current == 'true'
        run: |
          common_labels=(
            --label com.cloudspacelab.clearsight=true
            --label "org.opencontainers.image.revision=$RELEASE_SHA"
          )
          docker build "${common_labels[@]}" -f Dockerfile.api -t "clearsight-api:$RELEASE_SHA" .
          docker build "${common_labels[@]}" -f Dockerfile.worker -t "clearsight-worker:$RELEASE_SHA" .
          docker build "${common_labels[@]}" -f Dockerfile.web -t "clearsight-web:$RELEASE_SHA" .

      - name: Assemble release bundle
        if: steps.freshness.outputs.current == 'true'
        run: |
          release="$RUNNER_TEMP/release"
          install -d "$release/scripts" "$release/migrations"
          docker save \
            "clearsight-api:$RELEASE_SHA" \
            "clearsight-worker:$RELEASE_SHA" \
            "clearsight-web:$RELEASE_SHA" \
            -o "$release/images.tar"
          install -m 0644 deploy/compose.demo.yaml "$release/compose.demo.yaml"
          install -m 0755 deploy/scripts/migrate.sh "$release/scripts/migrate.sh"
          install -m 0755 deploy/scripts/release.sh "$release/scripts/release.sh"
          cp migrations/*.up.sql "$release/migrations/"
          printf 'false\n' > "$release/schema-backward-compatible"

      - name: Configure constrained SSH identity
        if: steps.freshness.outputs.current == 'true'
        env:
          DEPLOY_KEY: ${{ secrets.CLEARSIGHT_DEPLOY_KEY }}
          KNOWN_HOSTS: ${{ secrets.CLEARSIGHT_DEPLOY_KNOWN_HOSTS }}
        run: |
          install -d -m 0700 "$HOME/.ssh"
          printf '%s\n' "$DEPLOY_KEY" > "$HOME/.ssh/clearsight_deploy"
          printf '%s\n' "$KNOWN_HOSTS" > "$HOME/.ssh/known_hosts"
          chmod 0600 "$HOME/.ssh/clearsight_deploy" "$HOME/.ssh/known_hosts"

      - name: Deploy exact SHA
        if: steps.freshness.outputs.current == 'true'
        env:
          DEPLOY_HOST: ${{ secrets.CLEARSIGHT_DEPLOY_HOST }}
          DEPLOY_USER: ${{ secrets.CLEARSIGHT_DEPLOY_USER }}
        run: |
          tar -C "$RUNNER_TEMP/release" -cf - . | \
            ssh -i "$HOME/.ssh/clearsight_deploy" \
              -o BatchMode=yes \
              -o IdentitiesOnly=yes \
              "$DEPLOY_USER@$DEPLOY_HOST" "deploy $RELEASE_SHA"
```

- [ ] **Step 4: Run static tests and commit**

Run: `python -m unittest discover -s deploy/tests -p '*_test.py' -v`

```bash
git add .github/workflows/deploy-demo.yml deploy/tests/deployment_config_test.py
git commit -m "ci: deploy green main revisions to demo"
```

### Task 9: Document operation and run full local release gates

**Files:**
- Create: `docs/engineering/demo-deployment.md`
- Modify: `README.md`

- [ ] **Step 1: Write the operator guide**

Document the non-production boundary, existing-PostgreSQL ownership, required GitHub secrets (`CLEARSIGHT_DEPLOY_KEY`, `CLEARSIGHT_DEPLOY_KNOWN_HOSTS`, `CLEARSIGHT_DEPLOY_HOST`, `CLEARSIGHT_DEPLOY_USER`), first bootstrap, exact ports/paths, health commands, current/previous SHA files, migration checksum behavior, safe rollback limitations, certificate renewal and incident steps.

- [ ] **Step 2: Link it from README without changing readiness claims**

Add one entry under operational/developer documentation; retain the explicit statement that production object storage/scanning and other production gates are incomplete.

- [ ] **Step 3: Run every release gate on the exact head**

Run:

```bash
test -z "$(gofmt -l $(find cmd internal -name '*.go' -type f))"
go test -race -coverprofile=coverage.out ./...
go test -tags postgres ./...
go vet ./...
cd web && npm ci --no-audit --no-fund && npm run typecheck && npm test && npm run build
python -m unittest discover -s deploy/tests -p '*_test.py' -v
bash -n deploy/scripts/*.sh
git diff --check
```

Expected: every command exits 0 with no test failures.

- [ ] **Step 4: Commit documentation**

```bash
git add docs/engineering/demo-deployment.md README.md
git commit -m "docs: add demo deployment runbook"
```

### Task 10: Bootstrap the shared server without creating PostgreSQL

**Files:**
- Server-only state under `/opt/clearsight-grc`, `/etc/nginx/conf.d/clearsight.conf`, `/usr/local/sbin/clearsight-ci-entrypoint`, and one forced-command key entry.

- [ ] **Step 1: Capture a read-only before-state**

Record active containers, `systemctl is-active postgresql nginx docker`, database names/owners, Nginx config checksum/list, listeners, disk space and current public responses. Save no secrets.

- [ ] **Step 2: Generate scoped credentials locally**

Generate a new Ed25519 CI keypair and a 64-hex database password. Keep the private key only long enough to store it in GitHub Actions secrets; do not add it to Git.

- [ ] **Step 3: Run bootstrap with the supplied root PEM**

Copy only the reviewed bootstrap bundle to a temporary root-owned server directory, verify its SHA against the local files, execute `bootstrap-server.sh`, then remove the temporary bootstrap copy. The script may create only the ClearSight database/role and documented ClearSight/Nginx paths.

- [ ] **Step 4: Verify the existing PostgreSQL service was preserved**

Compare the before/after server version, PID/service state, database list and unrelated database owners. Confirm there is no ClearSight PostgreSQL container and no new `5432` Docker publication.

- [ ] **Step 5: Validate Nginx/TLS and forced-command restrictions**

Run `nginx -t`, curl the domain, verify the dedicated key rejects arbitrary commands/PTY/forwarding, and confirm it accepts only a syntactically valid `deploy <sha>` stream.

### Task 11: Configure GitHub, push, and verify the first automatic deployment

**Files:**
- GitHub repository Actions secrets and remote `main`.

- [ ] **Step 1: Configure repository secrets**

Store the dedicated private key, an `ssh-keyscan` result independently verified against the host key, host `139.162.40.237`, and user `root`. Never store the broad `bigbundle.pem` key.

- [ ] **Step 2: Re-run verification immediately before push**

Run the complete Task 9 gate again on the final commit and confirm `git status --short` is empty except expected committed work.

- [ ] **Step 3: Push the reviewed commits to `origin/main`**

Run: `git push origin main`

Expected: fast-forward push succeeds.

- [ ] **Step 4: Observe CI and deployment to a terminal state**

Wait for the exact pushed SHA's `CI` workflow to pass, then for `Deploy demo` to complete. Do not infer success from an older run.

- [ ] **Step 5: Perform public and server acceptance checks**

Verify:

```text
GET https://clearsight.cloudspacetechs.com/                 -> 200 and demo login
GET https://clearsight.cloudspacetechs.com/?demo=0          -> 200, live-preview notice, no demo-only controls after login
GET https://clearsight.cloudspacetechs.com/health/ready     -> 200 with PostgreSQL mode
```

Confirm the three ClearSight containers use the pushed SHA, API health is healthy, `state/current-sha` matches, migrations are recorded once, the native PostgreSQL service remains active, unrelated containers remain unchanged and no PostgreSQL container was created.

- [ ] **Step 6: Remove local transient credentials**

Delete the temporary dedicated private key only after GitHub confirms the secret is stored and a deployment succeeds. Preserve the user's supplied PEM at its original path untouched.
