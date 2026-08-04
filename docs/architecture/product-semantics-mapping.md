# Product Semantics to Architecture Mapping

This document maps the canonical user-facing operating model to ClearSight’s deeper architecture documents.

It exists because the architecture was initially described through the Institutional Risk Graph, Materiality Compiler, Living Evidence Fabric, Decision Ledger, and Governed AI Operators. Those remain useful internal mechanisms, but they are not the primary product language or mandatory navigation.

In case of semantic conflict, [`../product/operating-model.md`](../product/operating-model.md) controls the product-facing meaning.

---

# 1. Core rule

> **Architecture explains how ClearSight works. The operating model defines what the product means to users.**

The graph, evidence fabric, materiality service, decision ledger, workflows, and AI operators may span multiple internal modules. A user should ordinarily experience them through a bounded Risk Situation, focused Capture request, population worklist, decision review, or verified outcome.

---

# 2. Canonical mapping

| Product object | Internal architectural representation | Important boundary |
|---|---|---|
| Scope | Institution, legal entity, jurisdiction, business unit, service, process, branch, product, vendor, asset population, customer segment, and typed relationships | Active scope must be visible before action; graph structure must not become mandatory navigation |
| Exposure Pattern | Reusable risk-scenario template, causal pattern, control-failure mode, indicators, claims, obligations, controls, and verification patterns | A pattern is reusable across channels and is not itself an incident or active situation |
| Risk Situation | Versioned material item plus affected scope, current risk context, claims, evidence state, required handling, decision, action, and verification state | This is the primary user-facing aggregate; do not force module hopping |
| Claim | Living Evidence Fabric Claim | A precise statement for a scope, purpose, population, and period |
| Evidence Recipe | Claim-type evidence policy, required facts, acceptable sources, freshness, coverage, independence, contradiction, and approval rules | Must be explicit and versioned; not an opaque score |
| Observation | Depending on context: normalized Signal, Evidence Item, Evidence Assertion, test result, import row, system fact, or human assertion | All capture methods share provenance; observation is not automatically a verified fact |
| Conclusion | Claim Conclusion, assurance conclusion, risk-state conclusion, incident fact determination, or control-effectiveness conclusion | Must identify included/excluded evidence, contradiction, assumptions, authority, and valid period |
| Decision | Decision Ledger aggregate | Human authority remains distinct from AI recommendation or workflow state |
| Action | Action/remediation aggregate and external execution record | External completion is implementation state, not verified outcome |
| Verification Contract | Decision and remediation verification contract | Measures defined outcome criteria; does not overstate causal proof |
| Source Profile | Integration/source registry, trust metadata, freshness, mapping version, limitations, purpose, and health | Automation does not create authority by itself |

---

# 3. Risk Situation and Materiality Compiler

The term **material item** in earlier architecture documents should be interpreted as the materiality output used to create or update a **Risk Situation**.

A Risk Situation contains more than a materiality score or alert. It includes:

- active scope and period;
- exposure patterns;
- source observations;
- affected banking objects;
- what changed and why now;
- appetite and tolerance context;
- claims and evidence state;
- uncertainty and contradiction;
- required authority;
- current decision, action, and verification state;
- and history.

The Materiality Compiler should therefore emit structured situation input rather than a user-facing alert stream.

Materiality may:

- create a situation;
- update an existing situation;
- group or link observations;
- merge or propose splitting situations;
- change required handling;
- suppress executive visibility while preserving analyst access;
- or reopen a prior situation.

The compiler must separately represent:

- estimated exposure;
- evidence uncertainty;
- source and data-quality debt;
- decision relevance;
- and confidence.

---

# 4. Exposure Pattern and Risk Scenario

Earlier documents distinguish risk taxonomy from risk scenario. Retain that distinction.

Use:

- **taxonomy item** for classification;
- **Exposure Pattern** for a reusable causal or failure pattern;
- **Risk Situation** for the current bounded instance;
- **incident** for a governed determination that an event occurred.

Example:

- taxonomy: operational risk;
- exposure pattern: channel service unavailable beyond approved tolerance;
- risk situation: current POS processor degradation affecting selected merchants with stale failover proof;
- incident: formally declared outage meeting incident criteria.

Exposure Patterns may contain default claims, sources, controls, indicators, thresholds, and verification methods, but institution policy and scope determine their application.

---

# 5. Observation, Signal, Evidence Item, and Assertion

The product term **Observation** is the common capture contract.

An Observation may be represented internally as:

- a Signal when it may indicate a relevant change;
- an Evidence Item when an original artifact or governed source snapshot is preserved;
- an Evidence Assertion when a structured fact is extracted or confirmed;
- a test result;
- a source-health record;
- a reconciliation result;
- or a human assertion.

One captured artifact may produce multiple observations.

Example:

A branch ATM photograph may produce:

- original Evidence Item containing the image;
- assertion that serial number `ATM-99281` is visible;
- assertion that external seal appears present;
- assertion that branch signage is visible;
- metadata observation describing capture time;
- contradiction with the asset register.

The product must preserve the difference between:

- explicit source value;
- machine-extracted value;
- inferred candidate;
- human-confirmed value;
- and approved conclusion.

---

# 6. Evidence Recipe and sufficiency policy

The Living Evidence Fabric already defines multidimensional sufficiency. An Evidence Recipe is the product-facing, versioned policy binding that model to a claim type and materiality.

It should specify:

- required facts;
- acceptable sources;
- what each source is authoritative for;
- source limitations;
- population and coverage;
- effective period and freshness;
- independence;
- authenticity and integrity;
- contradiction policy;
- approval;
- and automated-evaluation permission.

A weighted summary may assist prioritization. Policy gates and dimension explanations remain authoritative.

---

# 7. Source Registry and integration trust

Earlier architecture documents include source trust, health, cursors, and provenance across several sections. These must be implemented as a coherent **Source Registry** product capability.

Every source profile includes:

- source and owner;
- collection method;
- authoritative facts;
- limitations;
- scope;
- identifiers;
- mapping version;
- expected freshness;
- current age and health;
- known data-quality limitations;
- unresolved mappings;
- access and purpose policy;
- and dependent conclusions.

Source health is not merely operational telemetry. Material source degradation can affect evidence sufficiency, situation confidence, decisions, reports, and assurance.

---

# 8. Institutional Risk Graph

The Institutional Risk Graph remains the shared time-aware relationship substrate.

Initial implementation should use:

- relational authoritative entities and typed relationships;
- append-only events and audit;
- optional rebuildable graph projection;
- optional search and vector projections;
- object storage for source artifacts.

User-experience rules:

- do not use the graph as default navigation;
- do not expose the full ontology to ordinary users;
- prefer readable paths, hierarchy, worklists, affected-scope summaries, and timelines;
- use node graphs only where spatial relationships improve understanding;
- enforce authorization on every node, edge, count, label, suggestion, and export.

---

# 9. Decision Ledger and Situation workspace

The Decision Ledger remains the authoritative decision record.

The user should ordinarily review it within the Risk Situation workspace:

```text
Summary
Evidence
Decision
Action
Outcome
History
```

Do not require a user to navigate to a separate Decision Ledger module to understand the situation.

The same applies to action, verification, and evidence architecture.

---

# 10. Governed AI Operators

Governed AI Operators remain internal constrained actors.

Product-facing principle:

> AI is a compiler from messy institutional inputs into proposed structured product objects and explanations.

AI may propose:

- observations;
- normalized identifiers;
- entity matches;
- claim mappings;
- contradictions;
- focused requests;
- situation summaries;
- options;
- and verification drafts.

AI must not become:

- the source of institutional truth;
- the evidence itself;
- the authority;
- the primary product navigation;
- or the only method of operation.

The interface should show what AI did only where material:

- source inputs;
- extracted and inferred values;
- confirmation state;
- uncertainty;
- and correction path.

---

# 11. Protected reporting boundary

Protected reporting should be treated as an isolated trust domain.

Recommended relationship:

```text
Protected Reporting System
├── protected case content
├── identity vault
├── investigator workspace
└── approved protected AI route

ClearSight ordinary operating model
└── receives only approved, minimized, sanitized observations or risk signals
```

Ordinary graph, search, analytics, source profiles, situation summaries, and executive briefs must not expose protected identity or identifying content.

---

# 12. Required architecture alignment tests

Architecture is conformant only when:

- a Risk Situation can be rendered without module hopping;
- a shallow regional-bank scope and deep multinational scope use the same model;
- forms, photos, spreadsheets, APIs, and telemetry all produce traceable observations;
- source authority and limitations are enforced;
- evidence recipes drive sufficiency;
- data-quality debt remains separate from risk severity;
- materiality updates situations rather than generating an unbounded alert stream;
- graph technology remains replaceable;
- AI output cannot bypass domain services;
- external task completion cannot close a situation;
- and point-in-time reconstruction includes sources, mappings, observations, conclusions, decisions, actions, and verification.