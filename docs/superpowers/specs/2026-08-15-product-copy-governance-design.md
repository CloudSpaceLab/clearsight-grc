# Product copy governance design

## Purpose

Prevent customer-facing language from drifting back into product commentary, implementation narration or generic AI-written guidance. ClearSight copy must help a bank stakeholder understand the object, current state, reason for attention, next action and consequence without explaining how the product was designed.

## Decision

Strengthen the root `AGENTS.md` as the normative writing gate for every future change. Keep the existing product-language audit brief as supporting rationale and the existing `copyQuality.test.ts` scan as automated enforcement.

The new rules will apply to all customer-visible language, including React components, server and API messages, onboarding, demo fixtures, empty states, errors, notifications, tooltips, labels, accessibility text and illustration descriptions.

## Required writing standard

Every customer-facing sentence must help with at least one of these needs:

- identify the business object or assigned task;
- state its current condition, source, owner, deadline or freshness;
- explain why the signed-in bank role needs to act;
- state the next action and its business result;
- explain a limitation, consequence or recovery step.

Copy must address the bank user directly at the point of work. It must not compare ClearSight with other product categories, defend a design decision, describe internal architecture or instruct users in product terminology they do not need for the task.

Examples of prohibited product narration include references to a “generic dashboard,” an “exact record,” canonical or bounded views, projections, authoritative servers, internal resolution behavior, implementation guarantees or equivalent rewordings. This is a semantic prohibition, not merely a fixed banned-word list.

## Interaction copy

- Headings name the task, record, state or decision in familiar banking language.
- Buttons use a direct verb and name the immediate result.
- Supporting text adds status, context, consequence or recovery information instead of repeating the heading.
- Acronyms familiar to the intended role, including CRO, CCO, CISO and GRC, retain their established casing.
- Guides remain concise, optional, dismissible, accessible and nonblocking. They orient users to work; they do not lecture users about ClearSight.
- Copy must preserve authority, evidence, legal-scope and uncertainty boundaries. Simpler language must not create a stronger compliance claim.

## Change workflow

Any change that adds or alters customer-facing copy must:

1. Review the complete affected workflow, not only the edited phrase.
2. Search every relevant copy source, including server responses and demo fixtures.
3. Add a new semantic pattern to `copyQuality.test.ts` when a newly observed class of product narration can be detected reliably without broad false positives.
4. Update focused workflow or component expectations where wording carries action, status or authority meaning.
5. Render and inspect each materially affected workspace at the relevant viewport sizes.
6. Confirm that guides and notices do not block primary work.

A phrase-by-phrase substitution is insufficient when equivalent product narration remains elsewhere in the workflow.

## Acceptance criteria

- The root `AGENTS.md` contains a clearly labelled mandatory customer-facing copy gate.
- The gate covers source scope, semantic admissibility, prohibited narration, acronym casing, onboarding behavior and verification.
- The rules point contributors to the existing automated scan without treating its pattern list as exhaustive.
- The new guidance does not weaken the existing authority, evidence, legal-scope, accessibility or degraded-path rules.
- Markdown formatting and repository checks pass.

## Out of scope

- Centralizing all copy into a catalogue.
- Rewriting interfaces that already passed the completed production audit.
- Changing API contracts, workflow commands or authorization behavior.
