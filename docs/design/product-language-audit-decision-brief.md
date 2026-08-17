# Product language audit decision brief

## Problem

Customer-facing screens contain sentences that explain ClearSight's design or implementation instead of helping a bank executive, risk leader, analyst or administrator complete work. Examples include comparisons with a "generic dashboard" and references to "exact records", "canonical" versions, "bounded" views, server responses and projections. This language adds reading burden, sounds like internal review commentary and can obscure the actual status or action.

## Decision

Rewrite visible interface text at its source: React screens, server-provided onboarding, demo fixtures and workflow messages. Every sentence must do at least one of the following:

- identify the object or task;
- state the current condition, source or time;
- explain why the signed-in user must act;
- state the next action, consequence or recovery step.

Use familiar operating nouns: Program, requirement, control, evidence, issue, approval, owner, due date and policy. Retain specialist terms such as OIDC, SCIM and authority policy only where an administrator needs them. Keep legal, evidence and approval limitations, but express them as direct consequences of the user's action.

Remove product defence, architecture narration and comparative commentary from customer-facing text. In particular, do not explain that the interface is not a dashboard or directory console, that a server is authoritative, or that a view is canonical, bounded or projected.

## Alternatives considered

1. **Source-wide rewrite with regression coverage — selected.** Corrects the language users actually encounter while preserving workflow semantics. A focused test prevents the known anti-patterns from returning.
2. **Phrase-by-phrase cleanup.** Lower immediate effort but leaves equivalent commentary elsewhere and does not address the underlying writing standard.
3. **Central copy catalogue refactor.** Could centralize future language management, but it would create a large structural change unrelated to the customer outcome and would not itself improve the writing.

## Safety boundaries

- Do not weaken authority, segregation-of-duties, evidence-quality or legal-scope warnings.
- Do not turn unknown or unavailable data into a positive claim.
- Do not change API contracts, workflow commands or authorization behavior.
- Empty states must name the population checked and the next valid action where one exists.

## Acceptance

- No customer-facing source includes the identified product-commentary phrases.
- Major workspaces remain understandable without implementation terminology.
- Existing workflow and component tests pass with updated expectations.
- Today, Programs, Work, Imports, Explore and Configure are rendered and reviewed at desktop and narrow widths.
