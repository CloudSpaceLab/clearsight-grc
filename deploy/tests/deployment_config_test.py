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

    def test_migrations_are_forward_only_and_checksum_guarded(self) -> None:
        script = self.read("deploy/scripts/migrate.sh")
        for value in ("ON_ERROR_STOP", "clearsight_schema_migrations", "sha256sum", "checksum mismatch",
                      "psql -X -1", "migration must have one outer BEGIN/COMMIT pair"):
            self.assertIn(value, script)
        self.assertIn(r"^[0-9]{6}_[a-z0-9_]+\.up\.sql$", script)
        self.assertNotIn(".down.sql", script)

    def test_ci_validates_only_the_current_forward_schema(self) -> None:
        workflow = self.read(".github/workflows/ci.yml")
        self.assertNotIn("rollback and reapply", workflow.lower())
        self.assertNotIn(".down.sql", workflow)
        self.assertIn("Verify deployment migration ledger", workflow)
        self.assertIn('go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/...', workflow)

    def test_forced_command_accepts_only_sha_deployments(self) -> None:
        script = self.read("deploy/scripts/ci-entrypoint.sh")
        for value in ("^deploy ([0-9a-f]{40})$", 'root=/opt/clearsight-grc', '"$root/incoming"',
                      "unsafe release path", "iflag=fullblock", 'test -f "$stage/scripts/verify-hosted-release.sh"',
                      '"$stage/scripts/release.sh" "$sha" "$stage"'):
            self.assertIn(value, script)
        self.assertNotIn('exec "$stage/scripts/release.sh"', script)

    def test_release_is_scoped_and_health_checked(self) -> None:
        script = self.read("deploy/scripts/release.sh")
        for value in ("5368709120", 'for component in api worker web', 'image="clearsight-$component:$sha"',
                      "scripts/migrate.sh", "compose -p clearsight", "13281/health/ready",
                      "13280/healthz", '"$release/scripts/verify-hosted-release.sh" "$sha"',
                      "state/current-sha", "com.cloudspacelab.clearsight=true"):
            self.assertIn(value, script)
        for forbidden in ("docker system prune", "docker volume prune", "down -v",
                          "systemctl restart postgresql"):
            self.assertNotIn(forbidden, script)

    def test_nginx_owns_only_the_clearsight_host(self) -> None:
        nginx = self.read("deploy/nginx/clearsight.conf")
        for value in ("server_name clearsight.cloudspacetechs.com", "127.0.0.1:13281",
                      "127.0.0.1:13280", "/etc/letsencrypt/live/clearsight.cloudspacetechs.com",
                      "proxy_no_cache 1", "client_max_body_size 21m"):
            self.assertIn(value, nginx)

    def test_bootstrap_preserves_shared_services(self) -> None:
        script = self.read("deploy/scripts/bootstrap-server.sh")
        for value in ("CREATE ROLE clearsight", "CREATE DATABASE clearsight OWNER clearsight",
                      "/opt/clearsight-grc", "/etc/nginx/conf.d/clearsight.conf", "nginx -t",
                      "host clearsight clearsight 127.0.0.1/32 scram-sha-256"):
            self.assertIn(value, script)
        for forbidden in ("DROP DATABASE", "DROP ROLE", "systemctl restart postgresql",
                          "docker system prune"):
            self.assertNotIn(forbidden, script)

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

    def test_hosted_verifier_is_exact_sha_read_only_and_redacted(self) -> None:
        verifier = self.read("deploy/scripts/verify-hosted-release.sh")
        for value in (
            'expected_sha="${1:?expected sha is required}"',
            "/health/ready",
            "/api/v1/demo/login",
            "/api/v1/session/status",
            "/api/v1/today",
            "/api/v1/forms/templates?limit=1",
            "/api/v1/vendors?limit=1",
            "/api/v1/evidence/access/start",
            '"revision"',
            '"authenticated"',
            "invalid_access_selector",
            "--retry-all-errors",
            "form_access_unavailable",
            '[[ "$denial_status" == 401 || "$denial_status" == 503 ]]',
        ):
            self.assertIn(value, verifier)
        self.assertNotIn("set -x", verifier)
        workflow = self.read(".github/workflows/deploy-demo.yml")
        self.assertIn('install -m 0755 deploy/scripts/verify-hosted-release.sh "$release/scripts/verify-hosted-release.sh"', workflow)
        self.assertNotIn('run: bash deploy/scripts/verify-hosted-release.sh "$RELEASE_SHA"', workflow)

    def test_email_readiness_requires_encryption_starttls_and_redacts_values(self) -> None:
        script = self.read("deploy/scripts/verify-email-readiness.sh")
        for value in (
            "CLEARSIGHT_RECIPIENT_KEYRING", "CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID",
            "CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY", "CLEARSIGHT_SMTP_HOST",
            "CLEARSIGHT_SMTP_PORT", "CLEARSIGHT_SMTP_USERNAME", "CLEARSIGHT_SMTP_PASSWORD",
            "CLEARSIGHT_SMTP_FROM", "CLEARSIGHT_SMTP_TLS_MODE", "STARTTLS",
            "/dev/tcp/", "openssl s_client", "-starttls smtp", "-verify_hostname",
            '"$CLEARSIGHT_SMTP_HOST"', "-verify_return_error",
        ):
            self.assertIn(value, script)
        for forbidden in ("env |", "printenv", "set -x", "echo $", "printf '%s' \"$"):
            self.assertNotIn(forbidden, script)
        hosted = self.read("deploy/scripts/verify-hosted-release.sh")
        self.assertIn('if [[ "${VERIFY_EMAIL_READINESS:-false}" == "true" ]]', hosted)
        self.assertIn('"$script_dir/verify-email-readiness.sh"', hosted)
        workflow = self.read(".github/workflows/deploy-demo.yml")
        self.assertIn('install -m 0755 deploy/scripts/verify-email-readiness.sh "$release/scripts/verify-email-readiness.sh"', workflow)

    def test_postgres_demo_is_seeded_and_artifacts_are_writable(self) -> None:
        dockerfile = self.read("Dockerfile.api")
        release = self.read("deploy/scripts/release.sh")
        compose = self.read("deploy/compose.demo.yaml")
        bootstrap = self.read("deploy/scripts/bootstrap-server.sh")
        self.assertIn("./cmd/seed-bank-reference", dockerfile)
        self.assertIn("/clearsight-seed-bank-reference", release)
        self.assertIn("-tenant 00000000-0000-4000-8000-000000000001", release)
        self.assertIn("-contributor 00000000-0000-4000-8000-000000000108", release)
        self.assertIn("/var/lib/clearsight/artifacts:Z", compose)
        self.assertEqual(compose.count("/var/lib/clearsight/artifacts:Z"), 2)
        self.assertIn("chown 65532:65532", bootstrap)
        self.assertIn("CLEARSIGHT_DEMO_TENANT_ID=00000000-0000-4000-8000-000000000001", bootstrap)
        foundation = self.read("deploy/scripts/seed-demo-foundation.sh")
        self.assertIn("INSERT INTO tenants", foundation)
        self.assertIn("INSERT INTO legal_entities", foundation)
        self.assertIn("INSERT INTO principals", foundation)
        for value in ("CLEARSIGHT-DEMO-AUTHORITY", "routing_policy_versions",
                      "refresh_effective_authority_routes", "demo-program-authorizer",
                      "ACCOUNTABLE_OWNER", "REVIEWER", "AUTHORIZER", "SIGNATORY",
                      "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER",
                      "00000000-0000-4000-8000-000000000104",
                      "00000000-0000-4000-8000-000000000106",
                      "d315abab6729fac5611327a56aa0f3d4ed07aad2ba160106beb0ce7a3f99e91e",
                      "157b7a984f7930c08002715ebc320f7dd1b0f2eb986cc03c18c7ff346065ce9f",
                      '"kind":"ROLE","ref":"EVIDENCE_RESPONDENT"',
                      "demo performer route does not resolve both governed assignees",
                      "definition IS DISTINCT FROM expected_definition"):
            self.assertIn(value, foundation)
        for value in ("INSERT INTO role_templates", "INSERT INTO org_positions",
                      "INSERT INTO position_role_bindings", "department_path",
                      "CLEARSIGHT_DEMO_STAFF_EMAIL", "INSERT INTO scim_sources",
                      "INSERT INTO scim_users", "demo-program-owner-contact",
                      "demo principal mappings differ from the managed fixture",
                      "demo role mappings differ from the managed fixture",
                      "demo position mappings differ from the managed fixture",
                      "demo position-role mappings differ from the managed fixture",
                      "actual.responsibilities IS DISTINCT FROM expected.responsibilities",
                      "actual.capabilities IS DISTINCT FROM expected.capabilities",
                      "actual.scope IS DISTINCT FROM expected.scope",
                      "actual.priority <> expected.priority"):
            self.assertIn(value, foundation)
        self.assertGreaterEqual(foundation.count("ON CONFLICT (id) DO UPDATE"), 7)
        self.assertNotIn("AND name IN ('ClearSight Demonstration Bank'", foundation)
        self.assertNotIn("AND name IN ('Demonstration Bank Nigeria'", foundation)
        self.assertNotIn("opatachibueze+staff", foundation)
        self.assertNotIn("demo_staff_email}'", foundation)
        self.assertIn('install -m 0700 "$stage/scripts/seed-demo-foundation.sh"', release)

    def test_go_image_tests_include_repository_contract_fixtures(self) -> None:
        for dockerfile_name in ("Dockerfile.api", "Dockerfile.worker"):
            dockerfile = self.read(dockerfile_name)
            self.assertIn("COPY api ./api", dockerfile)
            self.assertIn("COPY migrations ./migrations", dockerfile)
            self.assertIn("COPY docs/architecture/durable-schema-ownership.md ./docs/architecture/durable-schema-ownership.md", dockerfile)

    def test_worker_packages_pdf_extraction_and_remains_non_root(self) -> None:
        worker = self.read("Dockerfile.worker")
        ci = self.read(".github/workflows/ci.yml")
        for value in ("debian:12.12-slim", "poppler-utils", "USER 65532:65532"):
            self.assertIn(value, worker)
        self.assertIn("poppler-utils postgresql-client", ci)


if __name__ == "__main__":
    unittest.main()
