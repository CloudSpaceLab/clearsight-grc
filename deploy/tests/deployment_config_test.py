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
