# ClearSight Product Differentiation

This document defines what must make ClearSight recognizably different from traditional GRC suites, lightweight compliance products, generic AI assistants, workflow platforms, security consoles, and compliance automation engines.

It conforms to [`operating-model.md`](operating-model.md).

---

# 1. Positioning

ClearSight is:

> **A direct, bank-first GRC operating system that turns incomplete institutional information into understandable risk situations, finds or requests only the missing proof, routes authorized decisions, and verifies whether defined outcome criteria were achieved.**

ClearSight remains capable of supporting risks, controls, obligations, policies, incidents, assets, vendors, evidence, decisions, actions, audit, assurance, and reporting.

The differentiation is not the presence of these records. It is that users do not have to operate them as disconnected modules.

A branch manager should see the exact ATM or operational facts requiring confirmation. A channel owner should see the POS or payments situation, affected population, evidence weakness, and next handling step. A CRO, CCO, or CISO should see only material situations, uncertainty, options, authority, and outcome state.

---

# 2. The product moat

The moat is the coherent interaction of the following mechanisms.

## 2.1 Direct banking situations

ClearSight presents a bounded situation rather than forcing users to reconstruct one from risk, control, incident, asset, vendor, evidence, and task modules.

A situation answers:

- what is happening;
- what changed;
- why it matters now;
- what is affected;
- which exposure patterns apply;
- what is known, stale, missing, or contradictory;
- what handling is required;
- who has authority;
- and how the response will be verified.

### Differentiating requirement

A user can understand and handle one situation from one workspace while every underlying record remains traceable.

## 2.2 Universal exposure patterns

ClearSight uses reusable ways banking operations can fail or cause harm:

- availability and resilience;
- asset and inventory integrity;
- identity and access;
- transaction integrity;
- reconciliation and settlement;
- fraud and abuse;
- data and privacy;
- customer and conduct harm;
- third-party and concentration dependency;
- change and configuration integrity;
- physical and environmental integrity;
- regulatory or policy non-conformance;
- model or automated-decision failure;
- and evidence or data-quality uncertainty.

These patterns apply across ATM, POS, mobile, branch, cards, payments, lending, treasury, technology, vendors, and other bank contexts.

### Differentiating requirement

ClearSight does not create a new risk architecture for every channel. Channel packs configure scopes, claims, sources, controls, indicators, and evidence recipes on top of a shared operating model.

## 2.3 Progressive integration with explicit source trust

ClearSight must work through:

- structured forms and controlled values;
- photographs and scans;
- spreadsheet and CSV upload;
- scheduled managed imports;
- APIs;
- and event or telemetry streams.

Every source has a governed profile describing:

- what it is authoritative for;
- what it cannot prove;
- owner;
- scope;
- expected and current freshness;
- identifiers and mapping version;
- health;
- known limitations;
- and unresolved records.

### Differentiating requirement

A regional bank can begin without perfect APIs, while a multinational bank can deepen automation without changing product semantics.

An automated source is not trusted merely because it is automated.

## 2.4 Claim-centric evidence recipes

Evidence is assembled around a precise claim, population, purpose, scope, and period.

An Evidence Recipe defines:

- required facts;
- acceptable source types;
- source-authority limits;
- coverage;
- freshness;
- independence;
- contradiction policy;
- and review requirement.

The system searches authorized existing observations first, identifies unresolved facts, selects the best source, and asks the smallest useful question.

### Differentiating requirement

Human evidence collection is a first-class sensing capability, but humans are contacted only where machine, imported, or existing evidence cannot establish the required fact.

## 2.5 Normalized multimodal observations

Photos, spreadsheets, forms, dropdowns, APIs, database records, telemetry, documents, messages, and attestations become normalized observations with:

- subject;
- asserted or observed fact;
- source;
- scope;
- effective and capture time;
- provenance;
- original artifact;
- transformation history;
- authority and limitation;
- sensitivity;
- and confirmation state.

### Differentiating requirement

Capture flexibility does not weaken provenance.

A spreadsheet row remains traceable to file, sheet, row, mapping version, and import. A photo remains linked to the original image, extracted region, model version, and human confirmation.

## 2.6 Data-quality transparency and contradiction

ClearSight explicitly represents:

- unresolved identity matches;
- duplicate identifiers;
- stale sources;
- incomplete populations;
- conflicting ownership;
- partially accepted imports;
- contradictory observations;
- and unavailable source systems.

Evidence debt and data-quality debt are not automatically treated as certain risk failure, but they reduce confidence and may increase governance urgency.

### Differentiating requirement

The product never hides uncertainty behind an integration-success badge, a completion percentage, or an opaque score.

## 2.7 Decision memory

Material decisions are durable institutional objects.

A decision preserves:

- situation and scope;
- what was known at the time;
- evidence used and excluded;
- uncertainty and contradiction;
- options and trade-offs;
- authority and segregation of duties;
- rationale;
- dissent and override;
- conditions;
- expiry and review triggers;
- actions;
- and verification method.

### Differentiating requirement

The institution can reconstruct why a decision was reasonable from the information available at the time rather than judging it only from later events.

## 2.8 Outcome-verified remediation

ClearSight separates:

- work planned;
- work completed;
- implementation evidence;
- outcome evidence;
- control conclusion;
- risk conclusion;
- and accepted verified result.

A verification contract defines observable criteria, source, population, baseline, threshold, observation period, authority, and failure response.

### Differentiating requirement

A ticket, policy, training, configuration change, or document upload cannot by itself produce verified green.

ClearSight verifies whether defined criteria were achieved; it does not overstate causal certainty.

## 2.9 Governed AI as a compiler

AI converts messy institutional content into proposed structured observations, mappings, claims, contradictions, questions, summaries, and options.

Every material AI capability has:

- identity and purpose;
- scope and data-class limit;
- approved models and tools;
- structured output;
- source lineage;
- confidence dimensions;
- authorization and approval gates;
- evaluation;
- monitoring;
- and audit.

### Differentiating requirement

AI reduces assembly work but never becomes the authority, the evidence, or the only interface.

## 2.10 Calm task-oriented interface

The primary surfaces are:

- Today;
- Situation;
- Capture;
- Explore;
- Configure.

The interface uses the correct visual form:

- cards for a small attention queue;
- tables for populations;
- comparisons for contradiction and reconciliation;
- step flows for capture and import;
- paths for dependencies;
- timelines for history;
- and charts for specific decision questions.

### Differentiating requirement

Internal graph, evidence, decision-ledger, workflow, and AI architecture must not become mandatory navigation.

---

# 3. Differentiation from product categories

## 3.1 Traditional enterprise GRC suites

Traditional suites provide broad records, workflow, configuration, reporting, and mature enterprise controls.

ClearSight should learn from their governance depth while avoiding:

- module silos;
- form-first configuration;
- long custom-schema programmes;
- static assessment campaigns;
- dashboards as the primary interface;
- repeated evidence requests;
- and workflow completion being treated as assurance.

ClearSight differentiates through situations, evidence recipes, source authority, progressive integration, contradiction, decision memory, and outcome verification.

## 3.2 Lightweight GRC and compliance products

Modern lightweight products show that GRC can be direct and usable.

ClearSight preserves that directness while supporting:

- multi-entity and jurisdictional banks;
- relationship- and purpose-aware authorization;
- large operational populations;
- data-quality reconciliation;
- materiality and appetite;
- protected reporting;
- three-lines independence;
- examiner-grade history;
- and flexible deployment.

## 3.3 Compliance automation engines

Probo or another execution engine may manage commodity framework controls, measures, recurring evidence collection, tasks, policies, vendors, audits, and documents.

ClearSight remains responsible for:

- banking situation context;
- exposure and materiality;
- source authority;
- protected evidence;
- decision authority;
- cross-domain relationships;
- contradiction;
- and outcome verification.

External completion becomes an observation or implementation state, not automatic risk closure.

## 3.4 Generic AI assistants

A generic assistant may summarize or search but usually lacks:

- durable authoritative state;
- object- and relationship-level authorization;
- temporal institutional context;
- source authority and mapping health;
- decision authority;
- verified side effects;
- calibrated abstention;
- and outcome responsibility.

ClearSight is not chat over GRC data. Conversation is one governed entry point into the same structured operating model.

## 3.5 Workflow and ticketing platforms

Workflow systems coordinate tasks but do not normally understand:

- why the work exists;
- which exposure, claim, service, customer, or obligation it affects;
- what evidence is sufficient;
- who may decide;
- and how the outcome should be verified.

ClearSight may use those systems for execution while retaining the institutional meaning.

## 3.6 Security, observability, fraud, and monitoring platforms

Specialist platforms produce high-value signals and technical evidence.

ClearSight does not replace them. It adds:

- service, customer, legal-entity, and vendor context;
- source-health and data-quality interpretation;
- appetite and authority;
- evidence claims;
- decision handling;
- and outcome verification.

## 3.7 Complaints, ethics, and whistleblower systems

Specialist systems may remain the primary case-management tool.

ClearSight’s value is connecting validated, minimized signals to risk situations, services, controls, customer harm, vendors, incidents, and remediation without weakening protected identity or investigation confidentiality.

---

# 4. Bank-size adaptability

## 4.1 Regional bank

May begin with:

- shallow institution hierarchy;
- channel and branch packs;
- spreadsheets;
- mobile photo capture;
- forms and controlled values;
- selected API connections;
- simple authority structure.

## 4.2 National bank

May add:

- multiple regions and business units;
- richer channel populations;
- event-driven sources;
- centralized source registry;
- larger appetite and committee structures;
- broader assurance and audit.

## 4.3 Multinational group

May add:

- legal entities and jurisdictions;
- group and local policies;
- data residency and model routing;
- entity-specific control implementations;
- delegated and group authority;
- cross-border vendors and concentration;
- more complex point-in-time reconstruction.

The product semantics remain unchanged. Complexity appears through scope and policy, not separate product forks.

---

# 5. What ClearSight owns, orchestrates, and consumes

## ClearSight owns

- Scope and institution context
- Exposure Pattern library and channel packs
- Risk Situations
- Source Registry and source authority
- Claims and Evidence Recipes
- Normalized Observations and evidence lineage
- Contradiction and data-quality debt
- Materiality and appetite interpretation
- Decision authority and Decision Ledger
- Verification Contracts and outcome state
- Situation-first interface
- Governed AI policy
- Point-in-time institutional memory
- Protected-report integration boundary

## ClearSight orchestrates

- compliance tasks;
- ITSM and project actions;
- evidence requests;
- vendor evidence exchange;
- incident and security remediation;
- identity changes;
- policy publication;
- audit requests;
- and regulatory-response preparation.

## ClearSight consumes

- core banking and payment platforms;
- transaction, fraud, and AML platforms;
- switch and channel telemetry;
- IAM, HR, CMDB, ITSM, ERP, procurement, and CRM;
- security and observability platforms;
- complaints and case systems;
- regulatory content;
- documents, spreadsheets, forms, and field observations.

---

# 6. Experience differentiation

## 6.1 Familiar situation language

The user sees ATM, POS, payments, branch, merchant, vendor, access, customer, resilience, and operational language before abstract GRC codes.

## 6.2 One situation workspace

Summary, evidence, decision, action, outcome, and history remain together.

## 6.3 No dead-end indicator

Every material state has a clear path to evidence, investigation, decision, action, monitoring, or verification.

## 6.4 Correct density

Executives see a few situation cards. Operational teams receive tables, worklists, import tools, and reconciliation views.

## 6.5 Evidence in context

Evidence appears beside the claim, population, source authority, conclusion, decision, and outcome it affects.

## 6.6 AI without theatre

AI is visible through reduced effort and structured assistance, not persistent chat bubbles or decorative sparkle controls.

## 6.7 Verified green

Green is earned only by accepted outcome evidence.

---

# 7. Differentiation tests

## Situation

- Can the user understand the bank situation without navigating several modules?
- Is the active scope and period clear?
- Does the situation connect all relevant records without exposing architecture?

## Exposure

- Does the feature reuse a universal exposure pattern?
- Can it apply to more than one banking channel where appropriate?
- Are channel-specific details configuration rather than a product fork?

## Integration and data quality

- Can the workflow begin with a spreadsheet or form and later progress to API or event data?
- Is source authority explicit?
- Are stale, partial, unresolved, and contradictory records visible?

## Evidence

- Does it improve a claim rather than merely collect a file?
- Are existing observations searched before asking a person?
- Is the smallest unresolved question asked?
- Can machine and human evidence disagree visibly?

## Decision and outcome

- Is authority explicit?
- Are options and important trade-offs visible?
- Are conditions and expiry preserved?
- Is implementation separate from verification?
- Are defined outcome criteria measurable?

## AI

- Is the AI capability constrained, grounded, structured, correctable, and able to abstain?
- Are explicit, extracted, inferred, confirmed, and approved values distinguishable?

## Experience

- Is the correct form used: card, table, capture step, comparison, path, timeline, or chart?
- Does the screen reduce default complexity?
- Is the next handling step obvious?
- Does light mode, dark mode, accessibility, localization, and performance remain strong?

## Product moat

- Could the feature exist unchanged in a generic GRC product?
- Does it strengthen direct situations, evidence recipes, source trust, contradiction, decision memory, or outcome verification?
- Does it complete part of the Observe → Situation → Evidence → Decide → Act → Verify loop?

Generic but necessary features should remain supporting capabilities and must not be presented as the moat.

---

# 8. Product principle summary

ClearSight wins when it:

- speaks the bank’s operational language;
- understands scope and source authority;
- turns fragmented data into bounded situations;
- works before perfect integrations exist;
- asks staff fewer and better questions;
- exposes contradiction instead of hiding it;
- preserves material decisions;
- distinguishes implementation from outcome;
- governs AI more rigorously than generic agents;
- and gives each stakeholder only the complexity required for their role.

The final product should not merely look futuristic.

It should make the institution’s risk programme feel continuously aware, directly understandable, minimally demanding, explicitly governed, and evidentially defensible.