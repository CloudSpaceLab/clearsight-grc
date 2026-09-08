"""Execute deployment verifiers in Bash with isolated, network-free adapters.

Run with Linux/WSL Python and Bash; missing tools fail rather than skip coverage.
Only timeout, curl and Docker effects are replaced. Python validation and Bash
errexit behavior execute unchanged, including the nested hosted email verifier.
"""

import base64
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
SHA = "a" * 40
WARNING = "warning: SMTP connectivity unavailable; email delivery readiness remains unverified"
ADAPTER = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ["TEST_CALLS"], "a", encoding="utf-8") as log:
    log.write(json.dumps([name, args]) + "\n")
def reject():
    print("unexpected test adapter invocation", file=sys.stderr)
    sys.exit(99)
if name == "timeout":
    if args[:3] == ["10", "bash", "-c"]:
        assert len(args) == 7 and "/dev/tcp/" in args[3]
        assert args[5:] == [os.environ["CLEARSIGHT_SMTP_HOST"], os.environ["CLEARSIGHT_SMTP_PORT"]]
        code = int(os.environ.get("TEST_TCP_EXIT", "0"))
    elif args[:3] == ["15", "openssl", "s_client"]:
        assert args[3:] == ["-connect", os.environ["CLEARSIGHT_SMTP_HOST"] + ":587",
            "-servername", os.environ["CLEARSIGHT_SMTP_HOST"],
            "-verify_hostname", os.environ["CLEARSIGHT_SMTP_HOST"],
            "-verify_return_error", "-starttls", "smtp"]
        code = int(os.environ.get("TEST_TLS_EXIT", "0"))
    else:
        reject()
    # Prove that both stdout and stderr from provider probes are suppressed.
    diagnostic = "provider-response-canary " + " ".join(v for k, v in os.environ.items() if k.startswith("CLEARSIGHT_"))
    print(diagnostic)
    print(diagnostic, file=sys.stderr)
    sys.exit(code)
elif name == "docker":
    if args == ["ps", "-q", "--filter", "label=com.cloudspacelab.clearsight=true",
                "--filter", "ancestor=clearsight-worker:" + os.environ["TEST_SHA"]]:
        if os.environ.get("TEST_WORKER", "running") not in ("missing", "unowned", "wrong_image"):
            print("owned-worker-id")
    elif len(args) == 4 and args[:2] == ["inspect", "-f"] and args[3] == "owned-worker-id":
        if args[2] == "{{.State.Status}}":
            print("exited" if os.environ.get("TEST_WORKER") == "stopped" else "running")
        elif args[2] == '{{ index .Config.Labels "org.opencontainers.image.revision" }}':
            print("b" * 40 if os.environ.get("TEST_WORKER") == "wrong_revision" else os.environ["TEST_SHA"])
        else:
            reject()
    else:
        reject()
elif name == "curl":
    url = args[-1]
    path = url.split("://", 1)[-1].partition("/")[2]
    if path == os.environ.get("TEST_CURL_FAIL_PATH"):
        sys.exit(22)
    if path == "health/ready":
        result = {"mode": "postgres", "revision": os.environ["TEST_SHA"], "status": "ready"}
        case = os.environ.get("TEST_API")
        if case == "unavailable":
            sys.exit(7)
        if case == "wrong_revision":
            result["revision"] = "b" * 40
        elif case == "wrong_mode":
            result["mode"] = "memory"
        elif case == "not_ready":
            result["status"] = "starting"
        elif case == "extra_field":
            result["extra"] = True
        print(json.dumps(result))
    elif path == "api/v1/demo/login":
        print('{}')
    elif path == "api/v1/session/status":
        print(json.dumps({"authenticated": os.environ.get("TEST_SESSION", "true") == "true"}))
    elif path in ("api/v1/today", "api/v1/forms/templates?limit=1", "api/v1/vendors?limit=1"):
        print('{"items": []}' if path != os.environ.get("TEST_BAD_READ") else '{"items": null}')
    elif path == "api/v1/evidence/access/start":
        status = os.environ.get("TEST_DENIAL_STATUS", "401")
        result = {"error": "form_access_failed" if status == "401" else "form_access_unavailable"}
        if os.environ.get("TEST_DENIAL_BAD_ERROR"):
            result["error"] = "other"
        if os.environ.get("TEST_DENIAL_LEAK"):
            result[os.environ["TEST_DENIAL_LEAK"]] = "must remain private"
        pathlib.Path(args[args.index("-o") + 1]).write_text(json.dumps(result), encoding="utf-8")
        print(status, end="")
    else:
        reject()
elif name == "sleep":
    assert args == ["2"]
else:
    reject()
'''


class EmailReadinessTests(unittest.TestCase):
    def setUp(self):
        self.assertIsNotNone(shutil.which("bash"), "Run this suite with Linux/WSL Python and Bash")
        self.assertIsNotNone(shutil.which("python3"), "python3 must be available to Bash")
        self.temp = tempfile.TemporaryDirectory(prefix="clearsight-email-test-")
        self.addCleanup(self.temp.cleanup)
        directory = Path(self.temp.name)
        self.calls_path = directory / "calls.jsonl"
        adapters = directory / "bin"
        adapters.mkdir()
        for name in ("timeout", "curl", "docker", "sleep", "openssl"):
            adapter = adapters / name
            adapter.write_text(ADAPTER, encoding="utf-8", newline="\n")
            adapter.chmod(0o755)
        # Normalize checked-out CRLF without altering any script behavior.
        self.scripts = directory / "scripts"
        self.scripts.mkdir()
        for name in ("verify-email-readiness.sh", "verify-hosted-release.sh"):
            target = self.scripts / name
            target.write_text((ROOT / "deploy/scripts" / name).read_text(encoding="utf-8"), encoding="utf-8", newline="\n")
            target.chmod(0o755)
        key = base64.b64encode(b"\x11" * 32).decode()
        hmac = base64.b64encode(b"\x22" * 32).decode()
        self.env = {
            "PATH": str(adapters) + os.pathsep + os.defpath,
            "TEST_CALLS": str(self.calls_path), "TEST_SHA": SHA,
            "CLEARSIGHT_RECIPIENT_KEYRING": json.dumps({"test-key": key}),
            "CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID": "test-key",
            "CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY": hmac,
            "CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL": "https://capture-canary.invalid",
            "CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED": "true",
            "CLEARSIGHT_SMTP_HOST": "smtp-host-canary.invalid", "CLEARSIGHT_SMTP_PORT": "587",
            "CLEARSIGHT_SMTP_USERNAME": "smtp-username-canary", "CLEARSIGHT_SMTP_PASSWORD": "smtp-password-canary",
            "CLEARSIGHT_SMTP_FROM": "sender-canary@example.invalid", "CLEARSIGHT_SMTP_TLS_MODE": "STARTTLS",
            "CLEARSIGHT_SMTP_SECRET_REF": "env:CLEARSIGHT_SMTP_PASSWORD",
        }

    def run_verifier(self, *, args=None, hosted=False, changes=None):
        self.calls_path.write_text("", encoding="utf-8")
        env = {**self.env, **(changes or {})}
        name = "verify-hosted-release.sh" if hosted else "verify-email-readiness.sh"
        result = subprocess.run([shutil.which("bash"), str(self.scripts / name), *(args if args is not None else [SHA])],
                                env=env, capture_output=True, text=True, timeout=10)
        self.calls = [json.loads(line) for line in self.calls_path.read_text(encoding="utf-8").splitlines()]
        self.assertNotIn("unexpected test adapter invocation", result.stdout + result.stderr)
        return result

    def assert_redacted(self, result):
        output = result.stdout + result.stderr
        for key in ("CLEARSIGHT_RECIPIENT_KEYRING", "CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY",
                    "CLEARSIGHT_SMTP_HOST", "CLEARSIGHT_SMTP_USERNAME", "CLEARSIGHT_SMTP_PASSWORD",
                    "CLEARSIGHT_SMTP_FROM", "CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL"):
            self.assertNotIn(self.env[key], output)
        for value in json.loads(self.env["CLEARSIGHT_RECIPIENT_KEYRING"]).values():
            self.assertNotIn(value, output)
        self.assertNotIn("provider-response-canary", output)
        self.assertNotIn("delivery_verified=true", output)
        self.assertNotIn("smtp_authenticated=true", output)

    def test_success_reports_connectivity_and_mandatory_checks_in_both_modes(self):
        for args in ([SHA], [SHA, "--smtp-advisory"]):
            with self.subTest(args=args):
                result = self.run_verifier(args=args)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn("smtp_connectivity=available\n", result.stdout)
                self.assertNotIn("smtp_connectivity=unavailable", result.stdout)
                self.assertNotIn(WARNING, result.stderr)
                for check in ("smtp_configured", "starttls_required", "recipient_protection_configured",
                              "capture_origin_secure", "api_revision_matches", "worker_revision_matches"):
                    self.assertIn(check + "=true\n", result.stdout)
                self.assertEqual([call[1][0] for call in self.calls if call[0] == "timeout"], ["10", "15"])
                self.assert_redacted(result)

    def test_probe_failures_are_fatal_by_default_and_preserve_exit_status(self):
        for phase in ("TCP", "TLS"):
            for code in (1, 124):
                with self.subTest(phase=phase, code=code):
                    result = self.run_verifier(changes={f"TEST_{phase}_EXIT": str(code)})
                    self.assertEqual(result.returncode, code)
                    self.assertNotIn("smtp_connectivity=available", result.stdout)
                    self.assertFalse(any(call[0] in ("curl", "docker") for call in self.calls))
                    self.assert_redacted(result)

    def test_advisory_probe_failures_continue_required_checks_and_skip_tls_after_tcp_failure(self):
        for phase in ("TCP", "TLS"):
            for code in (1, 124):
                with self.subTest(phase=phase, code=code):
                    result = self.run_verifier(args=[SHA, "--smtp-advisory"], changes={f"TEST_{phase}_EXIT": str(code)})
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stderr, WARNING + "\n")
                    self.assertEqual(result.stdout.count("smtp_connectivity=unavailable\n"), 1)
                    self.assertNotIn("smtp_connectivity=available", result.stdout)
                    self.assertIn("api_revision_matches=true", result.stdout)
                    self.assertIn("worker_revision_matches=true", result.stdout)
                    self.assertEqual([call[1][0] for call in self.calls if call[0] == "timeout"], ["10"] if phase == "TCP" else ["10", "15"])
                    self.assert_redacted(result)

    def test_rejects_missing_malformed_unknown_and_excess_arguments(self):
        for args in ([], ["invalid"], [SHA, ""], [SHA, "--unknown"], [SHA, "--smtp-advisory", "extra"], [SHA, "--smtp-advisory", "--smtp-advisory"]):
            with self.subTest(args=args):
                result = self.run_verifier(args=args)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.calls, [])

    def test_all_required_configuration_remains_fatal_before_any_probe(self):
        required = [key for key in self.env if key.startswith("CLEARSIGHT_")]
        for key in required:
            with self.subTest(key=key):
                result = self.run_verifier(args=[SHA, "--smtp-advisory"], changes={key: ""})
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.calls, [])
                self.assert_redacted(result)

    def test_unsafe_configuration_and_bad_keys_are_fatal_before_any_probe(self):
        cases = {
            "CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED": ["false"],
            "CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL": ["http://capture-canary.invalid"],
            "CLEARSIGHT_SMTP_TLS_MODE": ["NONE", "TLS"],
            "CLEARSIGHT_SMTP_PORT": ["0", "65536", "abc", "587;exit 0"],
            "CLEARSIGHT_SMTP_FROM": ["invalid"], "CLEARSIGHT_SMTP_SECRET_REF": ["plaintext"],
            "CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID": ["missing-key"],
            "CLEARSIGHT_RECIPIENT_KEYRING": ["not-json", "[]", "{}", '{"test-key":"bad-base64!"}',
                                              '{"test-key":"eA=="}', json.dumps({str(n): "eA==" for n in range(9)})],
            "CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY": ["bad-base64!", "eA=="],
        }
        for key, values in cases.items():
            for value in values:
                with self.subTest(key=key, value=value):
                    result = self.run_verifier(args=[SHA, "--smtp-advisory"], changes={key: value})
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(self.calls, [])
                    self.assert_redacted(result)

    def test_advisory_warning_does_not_mask_later_api_failure(self):
        for case in ("wrong_revision", "wrong_mode", "not_ready", "extra_field", "unavailable"):
            with self.subTest(case=case):
                result = self.run_verifier(args=[SHA, "--smtp-advisory"], changes={"TEST_TCP_EXIT": "124", "TEST_API": case})
                self.assertIn(WARNING, result.stderr)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn("api_revision_matches=true", result.stdout)
                self.assert_redacted(result)

    def test_advisory_warning_does_not_mask_later_worker_failure(self):
        for case in ("missing", "stopped", "wrong_revision", "unowned", "wrong_image"):
            with self.subTest(case=case):
                result = self.run_verifier(args=[SHA, "--smtp-advisory"], changes={"TEST_TLS_EXIT": "1", "TEST_WORKER": case})
                self.assertIn(WARNING, result.stderr)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn("worker_revision_matches=true", result.stdout)
                self.assert_redacted(result)

    def test_hosted_advisory_outage_still_completes_login_reads_and_denial(self):
        for status in ("401", "503"):
            with self.subTest(status=status):
                result = self.run_verifier(hosted=True, changes={"VERIFY_EMAIL_READINESS": "true", "TEST_TCP_EXIT": "124", "TEST_DENIAL_STATUS": status})
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(WARNING, result.stderr)
                self.assertIn("smtp_connectivity=unavailable", result.stdout)
                self.assertIn("verified hosted release " + SHA, result.stdout)
                urls = [call[1][-1] for call in self.calls if call[0] == "curl"]
                for path in ("health/ready", "api/v1/demo/login", "api/v1/session/status", "api/v1/today",
                             "api/v1/forms/templates?limit=1", "api/v1/vendors?limit=1", "api/v1/evidence/access/start"):
                    self.assertTrue(any(url.endswith("/" + path) for url in urls), path)
                self.assert_redacted(result)

    def test_hosted_email_disabled_still_requires_exact_owned_running_worker(self):
        for case in ("running", "missing", "stopped", "wrong_revision", "unowned", "wrong_image"):
            with self.subTest(case=case):
                result = self.run_verifier(hosted=True, changes={"VERIFY_EMAIL_READINESS": "false", "TEST_WORKER": case,
                                                              "CLEARSIGHT_SMTP_PASSWORD": ""})
                self.assertEqual(result.returncode == 0, case == "running", result.stderr)
                self.assertFalse(any(call[0] == "timeout" for call in self.calls))
                self.assertTrue(any(call[0] == "docker" for call in self.calls))
                self.assertNotIn("smtp_connectivity=", result.stdout)

    def test_hosted_security_and_application_failures_stay_fatal_after_smtp_warning(self):
        cases = [{"TEST_CURL_FAIL_PATH": "api/v1/demo/login"}, {"TEST_SESSION": "false"},
                 {"TEST_DENIAL_STATUS": "200"}, {"TEST_DENIAL_BAD_ERROR": "true"}]
        cases += [{"TEST_BAD_READ": path} for path in ("api/v1/today", "api/v1/forms/templates?limit=1", "api/v1/vendors?limit=1")]
        cases += [{"TEST_DENIAL_LEAK": key} for key in ("distribution", "recipient", "audience_hint", "route_selector", "request_id")]
        cases += [{"TEST_API": "wrong_revision"}, {"TEST_WORKER": "wrong_revision"}]
        for case in cases:
            with self.subTest(case=case):
                result = self.run_verifier(hosted=True, changes={"VERIFY_EMAIL_READINESS": "true", "TEST_TCP_EXIT": "124", **case})
                self.assertIn(WARNING, result.stderr)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn("verified hosted release", result.stdout)
                self.assert_redacted(result)

    def test_hosted_advisory_does_not_ignore_invalid_recipient_protection(self):
        result = self.run_verifier(hosted=True, changes={"VERIFY_EMAIL_READINESS": "true", "CLEARSIGHT_RECIPIENT_KEYRING": "{}"})
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("verified hosted release", result.stdout)
        self.assertFalse(any(call[0] == "curl" for call in self.calls))


if __name__ == "__main__":
    unittest.main()
