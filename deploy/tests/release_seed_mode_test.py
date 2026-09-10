"""Exercise release seed selection with isolated local command adapters."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
SHA = "a" * 40
ADAPTER = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ["TEST_CALLS"], "a") as stream:
    stream.write(json.dumps([name, args]) + "\n")
if name == "df":
    print("Avail\n10737418240")
elif name == "curl":
    sys.exit(int(os.environ.get("TEST_HEALTH_EXIT", "0")))
elif name == "docker":
    if args[:2] == ["image", "inspect"]:
        print("true" if "com.cloudspacelab" in args[3] else os.environ["TEST_SHA"])
    elif args[:2] == ["inspect", "-f"]:
        print("healthy")
    elif "ps" in args:
        print("api-test-id")
    elif args[0] == "run":
        sys.exit(int(os.environ.get("TEST_SEED_EXIT", "0")))
    elif args[0] not in ("load", "compose", "image"):
        sys.exit(99)
elif name != "sleep":
    sys.exit(99)
'''


class ReleaseSeedModeTests(unittest.TestCase):
    def setUp(self):
        self.assertIsNotNone(shutil.which("bash"), "Run with Linux/WSL Python and Bash")
        self.temp = tempfile.TemporaryDirectory(prefix="clearsight-release-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.calls = self.root / "calls.jsonl"
        self.stage = self.root / "incoming" / (SHA + ".test")
        for path in (self.root / "config", self.root / "state", self.stage / "scripts",
                     self.stage / "migrations", self.root / "bin"):
            path.mkdir(parents=True, exist_ok=True)
        for name in ("df", "curl", "docker", "sleep"):
            target = self.root / "bin" / name
            target.write_text(ADAPTER, encoding="utf-8", newline="\n")
            target.chmod(0o755)
        for name in ("migrate.sh", "seed-demo-foundation.sh", "verify-hosted-release.sh",
                     "verify-email-readiness.sh"):
            target = self.stage / "scripts" / name
            target.write_text('#!/usr/bin/env bash\nset -e\nprintf \'["%s", []]\\n\' "${0##*/}" >> "$TEST_CALLS"\n'
                              + ('exit "${TEST_VERIFY_EXIT:-0}"\n' if name == "verify-hosted-release.sh" else ''),
                              encoding="utf-8", newline="\n")
            target.chmod(0o755)
        (self.stage / "compose.demo.yaml").touch()
        script = (ROOT / "deploy/scripts/release.sh").read_text(encoding="utf-8")
        script = script.replace("root=/opt/clearsight-grc", 'root="' + str(self.root) + '"')
        script = script.replace("lock=/run/lock/clearsight-deploy.lock", 'lock="' + str(self.root / "deploy.lock") + '"')
        self.script = self.root / "release.sh"
        self.script.write_text(script, encoding="utf-8", newline="\n")

    def run_release(self, mode=None, **changes):
        config = "" if mode is None else "CLEARSIGHT_DEMO_SEED_MODE=" + mode + "\n"
        (self.root / "config/app.env").write_text(config, encoding="utf-8")
        env = {"PATH": str(self.root / "bin") + os.pathsep + os.defpath,
               "TEST_SHA": SHA, "TEST_CALLS": str(self.calls), **changes}
        result = subprocess.run([shutil.which("bash"), str(self.script), SHA, str(self.stage)],
                                env=env, capture_output=True, text=True, timeout=15)
        self.invocations = [json.loads(line) for line in self.calls.read_text().splitlines()] if self.calls.exists() else []
        return result

    def test_manual_preserves_foundation_health_and_verification_without_reference_seeding(self):
        result = self.run_release("manual", TEST_SEED_EXIT="88")
        self.assertEqual(result.returncode, 0, result.stderr)
        names = [name for name, _ in self.invocations]
        for required in ("migrate.sh", "seed-demo-foundation.sh", "verify-hosted-release.sh", "curl"):
            self.assertIn(required, names)
        self.assertFalse(any(name == "docker" and args[0] == "run" for name, args in self.invocations))
        self.assertEqual((self.root / "state/current-sha").read_text().strip(), SHA)

    def test_reference_default_retains_seed_failure_gate(self):
        result = self.run_release(TEST_SEED_EXIT="88")
        self.assertEqual(result.returncode, 88, result.stderr)
        self.assertTrue(any(name == "docker" and args[0] == "run" for name, args in self.invocations))
        self.assertFalse((self.root / "state/current-sha").exists())

    def test_explicit_reference_seeds_and_promotes(self):
        result = self.run_release("reference")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(any(name == "docker" and args[0] == "run" for name, args in self.invocations))
        self.assertEqual((self.root / "state/current-sha").read_text().strip(), SHA)

    def test_unknown_mode_rejected_before_mutations(self):
        result = self.run_release("manul")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("CLEARSIGHT_DEMO_SEED_MODE", result.stderr)
        self.assertFalse(any(name != "df" for name, _ in self.invocations))
        self.assertFalse((self.root / "releases").exists())
        self.assertFalse((self.root / "state/current-sha").exists())

    def test_manual_verifier_failure_prevents_promotion(self):
        result = self.run_release("manual", TEST_VERIFY_EXIT="77")
        self.assertEqual(result.returncode, 77, result.stderr)
        self.assertFalse((self.root / "state/current-sha").exists())


if __name__ == "__main__":
    unittest.main()
