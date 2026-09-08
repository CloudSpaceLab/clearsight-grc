# SMTP Advisory Deployment Implementation Plan

> **For agentic workers:** Use subagent-driven-development for the shell/test task, followed by independent spec and quality review. Steps use checkbox syntax for tracking.

**Goal:** Allow application deployment during SMTP outages without hiding security or application failures.

**Architecture:** Add one explicit advisory argument to the existing email verifier; only network probe failures become warnings. Hosted verification requests that mode and independently requires the expected worker. Preserve strict standalone email acceptance and existing delivery runtime.

**Tech Stack:** Bash, Python standard-library unittest, existing GitHub Actions and Docker Compose.

## Task 1: Shell behavior and executable regression

Files: `deploy/scripts/verify-email-readiness.sh`, `deploy/scripts/verify-hosted-release.sh`, `deploy/tests/deployment_config_test.py`, new `deploy/tests/email_readiness_test.py`, `.github/workflows/ci.yml`.

- [x] Confirm existing isolated worktree; create `codex/smtp-advisory-deploy` from current main. Preserve unrelated PNG. Baseline: 16 deployment tests pass.
- [x] Add tests that run real verifier scripts using isolated fake `timeout`, `curl` and `docker` executables with public synthetic configuration. Assert TCP/TLS outage success only in advisory mode, strict-mode failure, redaction, unavailable status, and later hard failures. Test successful probes and hosted verification with email enabled/disabled. Reject malformed args, keys, unsafe transport/origin, missing/stopped/wrong worker and wrong/unavailable API. Keep adapters bounded and never contact SMTP in tests.
- [x] Run `python -m unittest discover -s deploy/tests -p '*test.py'`; observe failures for missing advisory behavior before changing shell code.
- [x] Add validated optional `--smtp-advisory` mode (default strict). Keep configuration and application checks as simple unguarded commands under `set -Eeuo pipefail`. Apply conditional handling only to the existing `timeout 10 ... /dev/tcp/` and `timeout 15 openssl ...` commands. On advisory outage print a fixed warning, unavailable status, and continue; strict outage returns the original nonzero status. Do not put the entire script in an `if`, `|| true`, or ignored-error subshell.
- [x] Add unconditional exact owned-worker verification to the hosted script and pass `--smtp-advisory` to its optional email check. Keep all hosted read/access checks mandatory.
- [x] Add `python3 -m unittest discover -s deploy/tests -p '*test.py'` to the existing backend CI job. Update structural expectations only for deliberate changes.
- [x] Run all deployment tests and `bash -n` for both verifiers; inspect red/green evidence and changed code. Commit only assigned source/tests/workflow files.

## Task 2: Documentation, review and release

Files: `docs/engineering/demo-deployment.md`, `docs/implementation-plan.md`, `docs/README.md`, this plan/design.

- [x] Document required versus advisory checks, default strict acceptance command, probe limitations, existing bounded retry/terminal reconciliation and no SMTP-based rollback. Link the approved design and retain historical email acceptance requirements.
- [x] Independent spec review, then quality review of the complete diff; resolve findings and repeat affected tests.
- [x] Run deployment tests, shell syntax, `git diff --check`, copy-quality regression and affected Go delivery/runtime tests. No UI code changes; retain full CI/UI gates and later hosted navigation evidence.
- [ ] Push PR linked to #147/#200. Wait for exact-head backend, web and UI checks; merge only green head. Let the existing current-main workflow deploy the new immutable SHA; do not modify the failed release directory or SMTP secrets.
- [ ] Verify deploy success, exact hosted SHA, owned worker running, strict SMTP probe result separately, and 12 hosted #201 navigation/focus/reflow checks. Inspect representative screenshots. Do not send mail or replay terminal jobs.
- [ ] Record exact results and remaining limits in #147/#200 and PR; leave broad acceptance issues open.

## Pre-release evidence

Code commit `6f6c128f9da7bbd08399ce767468991a6c058945`: test-first run produced 42 failing subcases across 28 tests with no test errors; the completed deployment suite passes all 28 tests under WSL. The parent and both independent reviewers reproduced the green result. Spec and quality reviews report no findings. LF Git blobs for both verifiers pass Bash syntax checks; the Windows working copy uses CRLF. Full `go test ./...`, fresh evidence/workflow/runtime tests and copy-quality regression pass. PR [#202](https://github.com/CloudSpaceLab/clearsight-grc/pull/202) retains exact-head CI and hosted release gates; final release receipts are recorded in [#147](https://github.com/CloudSpaceLab/clearsight-grc/issues/147), not inferred from this pre-release evidence.
