# ClearSight Product Differentiation

This document defines what must make ClearSight recognizably different from traditional GRC suites, lightweight compliance tools, generic AI assistants, workflow platforms, and compliance automation engines.

It is not a competitor comparison checklist. It is the product boundary that prevents ClearSight from becoming an undifferentiated collection of familiar GRC features.

---

# 1. Positioning

ClearSight is:

> **A bank-first, AI-native risk operating system that converts institutional signals into material decisions, captures the minimum necessary evidence from the best available sources, governs action, and verifies actual risk outcomes.**

The category distinction is important.

A conventional GRC system primarily records governance activity. ClearSight must actively reduce the effort and latency required to:

- detect meaningful change;
- understand cross-domain exposure;
- collect defensible evidence;
- make an authorized decision;
- coordinate action;
- verify effectiveness;
- and retain the institution’s reasoning over time.

---

# 2. The core product moat

ClearSight’s differentiation comes from the interaction of seven capabilities. Any one capability can be imitated. Their coherent combination is the product moat.

## 2.1 Institutional Risk Graph

The graph connects:

- objectives;
- legal entities;
- business units;
- products;
- customers and customer segments;
- critical services;
- processes;
- locations;
- people and accountable roles;
- systems and infrastructure;
- data assets;
- models and AI systems;
- vendors and fourth parties;
- contracts;
- regulations and obligations;
- policies;
- control objectives and implementations;
- risk scenarios;
- appetite statements and thresholds;
- signals;
- incidents, losses, complaints, and near misses;
- evidence and claims;
- findings and issues;
- decisions;
- actions;
- verification outcomes;
- and assurance conclusions.

The graph is not a visualization feature. It is the shared semantic substrate used by materiality, evidence requests, decisions, authorization, impact analysis, AI grounding, and assurance.

### Differentiating requirement

A change in one domain must propagate meaningfully across relevant relationships without duplicating data into every module.

Example:

A vendor’s resilience test expires. ClearSight should be able to determine that the vendor supports a payment service, the service has a strict impact tolerance, a related continuity finding remains open, the bank is within a peak transaction period, and the evidence supporting failover capability is now stale. The executive should receive one contextualized decision item rather than several disconnected alerts.

## 2.2 Materiality Compiler

Most enterprise systems collect more information than executives can use. ClearSight must compile large volumes of signals into a small number of defensible material changes.

The Materiality Compiler considers:

- appetite and limits;
- affected critical operations;
- customer scale and vulnerability;
- financial impact and loss potential;
- service disruption and recovery window;
- regulatory relevance and deadlines;
- legal-entity and jurisdiction exposure;
- risk velocity;
- concentration and propagation;
- evidence strength;
- control criticality;
- reversibility;
- management authority;
- and current institutional context.

### Differentiating requirement

The executive view is not sorted by severity alone. It is composed based on **decision relevance**.

A high-severity item that is already contained, well evidenced, and within delegated authority may be less important to a CRO than a moderate but fast-moving exposure with weak evidence and a pending board deadline.

## 2.3 Living Evidence Fabric

Traditional GRC evidence processes are document-centric and campaign-driven. ClearSight must be claim-centric and continuous.

The Living Evidence Fabric:

- identifies the exact claim needing proof;
- searches existing evidence before asking a person;
- determines what is missing;
- selects the best available source;
- generates the smallest useful request;
- captures original evidence through the user’s normal channel;
- evaluates relevance, authenticity, coverage, freshness, independence, completeness, consistency, reliability, and traceability;
- identifies contradiction;
- requests focused follow-up;
- and refreshes evidence based on risk rather than calendar alone.

### Differentiating requirement

Human evidence capture is not a secondary workflow. It is a first-class sensing capability spanning staff, vendors, customers, and confidential reporters.

## 2.4 Decision Ledger

Most systems preserve task and approval history but not the complete logic of a material decision.

The Decision Ledger preserves:

- what changed;
- affected institutional context;
- what was known;
- what remained uncertain;
- evidence used and excluded;
- available options;
- expected risk movement and cost;
- authority and segregation-of-duties checks;
- rationale;
- conditions and expiry;
- dissent and override;
- chosen action;
- and verification outcome.

### Differentiating requirement

The institution must be able to reconstruct a decision from the perspective of what was known at the time, rather than judging it only from later outcomes.

## 2.5 Outcome-Verified Remediation

Traditional issue management commonly closes when planned activities are completed.

ClearSight separates:

- action completion;
- implementation evidence;
- effectiveness evidence;
- control effectiveness;
- risk movement;
- and decision acceptance.

### Differentiating requirement

A material issue does not close merely because a new policy was published, a configuration changed, a training was delivered, or a ticket was completed. It closes only when the defined outcome is observed over the required period and accepted by the appropriate authority.

## 2.6 Governed AI Operators

A generic assistant can summarize, draft, and search. ClearSight operators must work as governed institutional actors.

Each operator has:

- identity;
- purpose;
- tenant and legal-entity scope;
- data classification limit;
- allowed tools;
- action classes;
- model-routing policy;
- confidence threshold;
- required approvals;
- and audit obligations.

### Differentiating requirement

Operator output is not merely logged. Every material action emits a structured record containing source lineage, model and policy version, rationale, confidence, authorization result, approval, execution result, and resulting domain changes.

## 2.7 Calm Risk Command Surface

The interface is designed around executive cognition and accountable decisions, not module navigation.

The interaction grammar is:

> **Brief → Explain → Act → Prove**

### Differentiating requirement

The default experience should feel simpler as institutional complexity increases. The system absorbs complexity and presents only the current decision surface, while preserving drill-down depth.

---

# 3. Differentiation from major product categories

## 3.1 Traditional enterprise GRC suites

Enterprise GRC suites provide broad workflow and record-management coverage, mature reporting, and extensive configuration. ClearSight should learn from their breadth without inheriting their interaction cost.

ClearSight must avoid:

- module silos;
- form-first configuration;
- long implementation programs caused by excessive custom schema work;
- static assessment campaigns;
- dashboard density as a proxy for sophistication;
- repeated human evidence requests;
- and workflow completion being treated as assurance.

ClearSight differentiates through:

- a shared time-aware graph;
- a materiality compiler;
- dynamic evidence capture;
- decision-centric executive interaction;
- outcome verification;
- protected external and human signals;
- and governed operators.

Archer Evolv provides useful reference points around connected risk intelligence, regulatory lineage, graph-based context, scenario simulation, and governed AI operators. ClearSight must go further in making **dynamic staff, customer, vendor, and whistleblower evidence capture** part of the primary operating model and in treating **verified outcome change** as the end of every material workflow.

Reference: <https://www.archerirm.com/evolv>

## 3.2 Lightweight and SMB-focused GRC products

Modern lightweight products demonstrate that GRC can be usable, fast, and direct. OpenGRC’s emphasis on “no bloat,” AI-assisted assessments, automated reporting, and straightforward project/risk management is a useful usability reference.

ClearSight must preserve that directness while supporting:

- multi-entity banks;
- strict authorization boundaries;
- complex dependency and concentration analysis;
- materiality and appetite;
- protected reporting;
- three-lines independence;
- examiner-grade lineage;
- high-volume signal ingestion;
- and bank-grade deployment and data residency.

Reference: <https://opengrc.com/product-1>

## 3.3 Compliance automation platforms

Compliance automation products are effective at framework control management, evidence collection, policy workflows, vendor records, audit preparation, and recurring compliance tasks.

Probo is a particularly useful open-source execution option. Its documented model includes frameworks, controls, measures, risks, vendors, assets, evidence, tasks, audits, documents, findings, obligations, snapshots, and MCP tools.

ClearSight must not recreate these functions without reason.

Instead:

- Probo or another engine may execute commodity compliance work.
- ClearSight determines materiality and institutional context.
- ClearSight governs actor authority.
- ClearSight reconciles automated evidence with human and external evidence.
- ClearSight owns protected reporting and sensitive bank workflows.
- ClearSight records material decisions.
- ClearSight verifies whether work actually changed risk.

References:

- <https://www.probo.com/docs>
- <https://www.probo.com/docs/getting-started/core-concepts>
- <https://www.probo.com/docs/api/mcp/overview>

## 3.4 Generic AI assistants

A generic assistant may answer a question about risk but usually lacks:

- institutional authority;
- durable domain state;
- object-level authorization;
- precise source lineage;
- temporal institutional context;
- workflow execution boundaries;
- calibrated abstention;
- and accountability for resulting action.

ClearSight is not “chat over GRC data.”

The conversational layer is one interface into a governed risk operating system. The same authorization, provenance, materiality, evidence, and decision rules apply whether an action begins through a chat command, a dashboard, an API, an event, or a scheduled workflow.

## 3.5 Workflow and ticketing platforms

Workflow systems coordinate tasks well but generally do not understand institutional risk semantics.

ClearSight should use them for execution while retaining:

- why the work exists;
- what risk and obligation it addresses;
- expected risk outcome;
- evidence requirements;
- authority;
- and verification status.

A ticket may be closed while the ClearSight issue remains open because the expected outcome has not been observed.

## 3.6 Security and monitoring platforms

Security systems produce high-value signals and technical evidence. ClearSight should not replace them.

ClearSight adds:

- business-service and customer context;
- risk-appetite interpretation;
- cross-domain dependency analysis;
- control and obligation linkage;
- executive materiality;
- decision authority;
- and remediation verification.

## 3.7 Complaints, ethics, and whistleblower systems

Specialist case systems manage reports and investigations. ClearSight can integrate with them, but its differentiating value is connecting protected and customer-originated signals to broader risk, controls, services, culture indicators, incidents, vendors, and remediation outcomes without weakening confidentiality.

---

# 4. Bank-first differentiation

ClearSight is not merely a generic enterprise product marketed to banks. Bank-specific requirements shape the underlying model.

## 4.1 Critical operations and resilience

The institutional graph must connect critical operations to the people, processes, technology, facilities, data, vendors, and fourth parties that enable them.

The product must support:

- impact tolerances;
- scenario testing;
- service disruption measurement;
- dependency concentration;
- stressed exit;
- recovery evidence;
- and board-level resilience oversight.

This aligns with the Basel Committee’s principles-based focus on governance, critical operations, dependency mapping, third-party dependency, incident management, continuity testing, and resilient ICT.

Reference: <https://www.bis.org/bcbs/publ/d516.htm>

## 4.2 Three-lines operation without data fragmentation

The same source evidence may inform first-line management, second-line challenge, and third-line assurance, but conclusions and authority must remain separate.

ClearSight must allow:

- one source fact;
- multiple independent conclusions;
- visible disagreement;
- and distinct approval or assurance authority.

The graph is shared. Judgment is not silently merged.

## 4.3 Multi-entity and jurisdictional context

Banks operate across legal entities, branches, products, regulators, data regions, and customer groups.

Every material object must support applicable scope. A control may be globally designed but differently implemented and evidenced by entity or jurisdiction.

## 4.4 Risk appetite as executable policy

Risk appetite cannot remain only in a board document.

ClearSight must convert appetite into governable, versioned decision logic while preserving qualitative judgment.

Examples:

- thresholds;
- escalation rules;
- authority limits;
- prohibited conditions;
- time-bound acceptance constraints;
- and evidence minimums.

## 4.5 Supervisory defensibility

The product must support point-in-time reconstruction of:

- applicable obligation;
- policy and control versions;
- evidence available;
- risk conclusion;
- AI involvement;
- human challenge and approval;
- action status;
- and later effectiveness.

---

# 5. Unique product mechanisms

## 5.1 Evidence debt

ClearSight should treat missing, stale, weak, or contradictory evidence as **evidence debt**.

Evidence debt is not the same as risk exposure, but it reduces confidence in the institution’s understanding and may itself require action.

Evidence debt can be measured by:

- material claims without sufficient evidence;
- evidence nearing expiry;
- self-attested claims lacking independent support;
- incomplete population coverage;
- unresolved contradictions;
- unavailable original sources;
- and conclusions relying on deprecated policies, controls, models, or mappings.

The executive view should surface evidence debt only where it changes a material decision or assurance statement.

## 5.2 Verification contracts

Every material remediation has a machine-readable verification contract.

Example:

```yaml
outcome: "Privileged access exceptions are removed or formally approved"
measure:
  source: "IAM directory snapshot"
  population: "Treasury Operations privileged roles"
  success: "0 unapproved active exceptions"
observation_period: "30 days"
required_evidence:
  - "daily IAM snapshots"
  - "approved exception records"
acceptance_authority: "Technology Risk Officer"
failure_action: "reopen issue and escalate to CISO"
```

This allows ClearSight to distinguish implementation from effectiveness.

## 5.3 Best-placed-source resolution

When evidence is missing, the system should rank possible sources based on:

- authority;
- directness;
- independence;
- freshness;
- coverage;
- accessibility;
- burden;
- cost;
- sensitivity;
- historical reliability;
- and conflict of interest.

The system should prefer machine evidence when it directly proves the claim, but it must not assume machine-generated evidence is always sufficient or unbiased.

## 5.4 Contradiction graph

Contradictory evidence should be represented explicitly rather than overwritten.

A contradiction record includes:

- the disputed claim;
- conflicting evidence versions;
- type of conflict;
- affected conclusions and decisions;
- urgency;
- assigned resolver;
- and resolution outcome.

## 5.5 Decision expiry

Risk acceptance and material decisions should rarely be indefinite.

ClearSight supports:

- expiry;
- review triggers;
- condition-based invalidation;
- appetite changes;
- evidence deterioration;
- incident realization;
- and business-context changes.

## 5.6 Counterfactual treatment view

The decision experience should compare plausible options:

- expected exposure change;
- confidence range;
- implementation time;
- cost;
- operational disruption;
- dependencies;
- reversibility;
- and residual uncertainty.

The system must clearly distinguish measured facts from modeled estimates.

## 5.7 Institutional judgment learning

The system may learn from expert corrections and outcomes, but it must not silently convert historical decisions into policy.

Learning inputs include:

- accepted and rejected recommendations;
- edited mappings;
- evidence sufficiency overrides;
- realized incidents and losses;
- verification outcomes;
- false-positive materiality decisions;
- and committee feedback.

Any learned policy change must be reviewable, versioned, and reversible.

---

# 6. Product boundaries

ClearSight should integrate rather than replace specialist systems when replacement does not strengthen the core moat.

## ClearSight owns

- institutional risk graph;
- risk appetite and materiality;
- living evidence fabric;
- evidence debt and contradiction;
- material decisions and authority;
- verification contracts;
- cross-domain risk propagation;
- protected reporting policy and identity isolation;
- executive risk command surface;
- governed operator registry and policy;
- assurance lineage;
- and point-in-time institutional memory.

## ClearSight orchestrates

- compliance automation;
- ITSM and project tasks;
- control evidence collection;
- vendor due diligence;
- incident response;
- security remediation;
- identity changes;
- policy publishing;
- audit requests;
- and regulatory response preparation.

## ClearSight consumes from specialist systems

- transaction monitoring;
- AML and fraud engines;
- SIEM, SOAR, EDR, DLP, IAM, and vulnerability platforms;
- core banking and payment platforms;
- HR, ERP, procurement, and CRM;
- complaints management;
- data catalog and CMDB;
- market and external threat intelligence;
- and regulatory content providers.

---

# 7. Experience differentiation

## 7.1 A calm first screen

The first screen is not a collection of modules and metrics. It is a role-specific brief showing:

- material change;
- evidence state;
- required authority;
- expected time-to-impact;
- and next decision.

## 7.2 No dead-end indicators

Every material indicator has one or more clear handling paths.

A user should never need to infer how to move from a chart to a governed action.

## 7.3 Evidence in context

Evidence is viewed beside the claim, conclusion, control, risk, and decision it supports—not in a detached repository.

## 7.4 AI appears as capability, not theater

AI should primarily be visible through reduced effort, better prefilled context, stronger explanations, and faster action.

Avoid unnecessary chat bubbles, “sparkle” buttons, and AI labels on ordinary deterministic functions.

## 7.5 Verified green

Green represents verified effectiveness or acceptable state supported by sufficient evidence. It must not represent:

- task completion;
- document presence;
- self-attestation alone;
- or absence of recent alerts.

---

# 8. Differentiation tests

A proposed feature should answer yes to the relevant questions.

## Evidence

- Does it improve a claim’s evidence rather than merely collect a file?
- Does it reduce the number of questions asked of a person?
- Can it reconcile machine and human evidence?
- Can it represent contradiction?

## Risk intelligence

- Does it use institutional relationships and materiality?
- Does it show why the change matters now?
- Can it explain propagation to customers, services, obligations, or entities?

## Decisions

- Does it identify the accountable authority?
- Does it present proportionate options and uncertainty?
- Does it preserve rationale and conditions?
- Does it define how success will be verified?

## AI

- Is the operator constrained and identifiable?
- Are sources and versions preserved?
- Can it abstain?
- Is a human required at the appropriate threshold?

## Experience

- Does it reduce default complexity?
- Is the primary next action obvious?
- Is the interface understandable without GRC jargon?
- Does it preserve visual calm and accessibility?

## Product moat

- Could this feature exist unchanged in any generic GRC product?
- Does it strengthen at least one ClearSight defining mechanism?
- Does it connect to the full Sense → Explain → Decide → Act → Prove → Learn loop?

If a feature is generic but necessary, implement it as a supporting capability and avoid presenting it as the product’s primary differentiation.

---

# 9. Product principle summary

ClearSight wins when it:

- understands the institution better than a form-based system;
- asks people less than a campaign-based system;
- connects more context than a module-based system;
- preserves more evidence than a chat-based system;
- governs AI more rigorously than a generic agent platform;
- requires stronger outcome proof than a task-management system;
- and gives executives less noise than a dashboard-centric system.

The final product should not merely look futuristic.

It should feel as though the institution’s risk program has become **continuously aware, minimally demanding, explicitly governed, and provably effective**.