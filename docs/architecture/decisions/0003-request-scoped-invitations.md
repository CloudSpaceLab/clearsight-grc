# ADR-0003 — Request-Scoped Invitations

**Status:** Accepted  
**Date:** 2026-08-04

## Context

ClearSight must collect focused evidence from employees, vendors, customers, and other external parties without granting broad tenant access. Permanent magic links or public forms would create replay, forwarding, leakage, and authorization risk.

## Decision

An invitation is a narrow capability bound to one request or approved request bundle.

Use:

- opaque cryptographically random tokens;
- token hashes at rest;
- request, audience, purpose, issue generation, expiry, usage, and revocation state;
- one-time or bounded token exchange for a short-lived server-side session;
- step-up identity verification based on sensitivity and consequence;
- token removal from the URL after exchange;
- content-minimized notifications and safe failure screens.

Invitation possession alone does not establish sufficient identity for high-impact or sensitive actions.

Protected anonymous reporting uses a separate identity-isolated mailbox and is not implemented through the ordinary invitation model.

## Consequences

External participants can complete narrow wizards without normal ClearSight accounts. The system must operate an invitation/session service, delivery audit, abuse controls, and recipient-resolution lifecycle.

## Guardrails

- no Matter browsing or general tenant access;
- no sensitive request data in URL, logs, referrers, page titles, analytics, or previews;
- forwarded, revoked, expired, replayed, and wrong-recipient links fail without metadata leakage;
- cancellation, recipient change, or request supersession invalidates prior invitations;
- final submit rechecks request, recipient, scope, and policy state;
- drafts and resume sessions remain request- and version-bound.

## Validation

Test issuance, delivery, redemption, step-up, resume, forwarding, replay, revocation, cancellation, wrong recipient, network interruption, and high-volume invitation bursts.

## Revisit when

Revisit identity and session methods when pilot data classes, customer channels, external federation, or regulatory requirements demand stronger or different assurance.
