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

    def test_forced_command_accepts_only_sha_deployments(self) -> None:
        script = self.read("deploy/scripts/ci-entrypoint.sh")
        for value in ("^deploy ([0-9a-f]{40})$", 'root=/opt/clearsight-grc', '"$root/incoming"',
                      "unsafe release path", 'exec "$stage/scripts/release.sh"'):
            self.assertIn(value, script)

    def test_release_is_scoped_and_health_checked(self) -> None:
        script = self.read("deploy/scripts/release.sh")
        for value in ("5368709120", 'for component in api worker web', 'image="clearsight-$component:$sha"',
                      "scripts/migrate.sh", "compose -p clearsight", "13281/health/ready",
                      "13280/healthz", "state/current-sha", "com.cloudspacelab.clearsight=true"):
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

    def test_postgres_demo_is_seeded_and_artifacts_are_writable(self) -> None:
        dockerfile = self.read("Dockerfile.api")
        release = self.read("deploy/scripts/release.sh")
        compose = self.read("deploy/compose.demo.yaml")
        bootstrap = self.read("deploy/scripts/bootstrap-server.sh")
        self.assertIn("./cmd/seed-bank-reference", dockerfile)
        self.assertIn("/clearsight-seed-bank-reference", release)
        self.assertIn("-tenant 00000000-0000-4000-8000-000000000001", release)
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
                      "definition IS DISTINCT FROM expected_definition"):
            self.assertIn(value, foundation)
        self.assertIn('install -m 0700 "$stage/scripts/seed-demo-foundation.sh"', release)

    def test_go_image_tests_include_repository_contract_fixtures(self) -> None:
        for dockerfile_name in ("Dockerfile.api", "Dockerfile.worker"):
            dockerfile = self.read(dockerfile_name)
            self.assertIn("COPY api ./api", dockerfile)
            self.assertIn("COPY migrations ./migrations", dockerfile)
            self.assertIn("COPY docs/architecture/durable-schema-ownership.md ./docs/architecture/durable-schema-ownership.md", dockerfile)


if __name__ == "__main__":
    unittest.main()
