# SMTP availability must not stop application deployment

Approved by the operator on 8 September 2026 after the #201 deployment failure; tracked under #147 and #200. This is a demo operations correction, not production email acceptance.

## Decision

Keep the existing bounded SMTP probe, but make connectivity advisory only when called by the hosted deployment verifier. Removing the probe loses operational evidence; a separate workflow adds unnecessary scheduling and configuration. No UI, database, runtime delivery or infrastructure configuration change is needed.

`verify-email-readiness.sh SHA` remains strict for explicit email acceptance. An explicit `--smtp-advisory` argument tolerates only a failed TCP or certificate-validated STARTTLS probe, prints a redacted warning and `smtp_connectivity=unavailable`, and continues to required checks. Success prints `smtp_connectivity=available`; neither result claims SMTP authentication, message delivery or inbox receipt. Unsupported arguments fail closed.

Required configuration, recipient encryption/HMAC, HTTPS capture origin, STARTTLS configuration, API exact revision and worker revision/running checks remain fatal. Hosted verification also checks the exact owned worker independently of whether optional email verification is enabled. Do not wrap the whole email verifier in a catch-all success handler: that would hide security and application failures.

Probe bounds remain TCP 10 seconds and STARTTLS 15 seconds, with the TLS probe skipped after TCP failure. SMTP diagnostics never log configuration values, credentials or provider responses, and never authenticate or send mail. Delivery remains subject to existing secure transport, durable receipts, bounded retry and operator reconciliation for unknown outcomes. No automatic replay of terminal failures.

## Acceptance

Execute the real shell scripts with controlled subprocess adapters: TCP timeout, TLS timeout/failure and successful probes; strict versus advisory mode; invalid arguments/configuration/key material; missing/stopped/wrong-revision worker; wrong/unavailable API; mandatory hosted login/read/access-denial checks after SMTP outage. Assert redacted output, no false connectivity success and nonzero exit for hard failures even after an advisory warning. Include the deployment suite in existing CI without adding dependencies or a workflow.

The normal immutable new-SHA deployment must succeed before release is claimed. Keep migrations, release locks, image ownership, current-main checks and rollback unchanged. Recheck exact hosted revision, worker state, login/core reads and the prepared #201 document-result focus checks. Prior failed CI and unresolved recipient journey issues remain historical truth.
