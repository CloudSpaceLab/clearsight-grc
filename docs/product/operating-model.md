# ClearSight Bank Operating Model

This document is the canonical product-semantic layer between the ClearSight vision and its deeper architecture.

It defines the smallest set of concepts through which ClearSight should make bank risk understandable, actionable, and evidentially defensible without exposing users to GRC architecture or requiring perfect enterprise integrations.

---

# 1. Product objective

ClearSight should help each stakeholder:

1. understand a bounded banking risk situation in familiar language;
2. see what is known, missing, stale, or contradictory;
3. provide or inspect only the evidence relevant to that situation;
4. make or route an authorized decision;
5. coordinate action;
6. verify whether the defined outcome criteria were achieved;
7. preserve the complete institutional history.

The product remains a comprehensive GRC platform, but users operate it through situations and tasks rather than modules and internal data models.

---

# 2. Canonical operating loop

```text
Observe
→ identify an exposure
→ create or update a risk situation
→ determine what must be true
→ find or request only missing proof
→ conclude what the evidence supports
→ route the required decision
→ act
→ verify the defined outcome
→ update the situation and institutional memory
```

This loop applies across operational risk, compliance, cyber, resilience, third-party risk, incidents, controls, audit, customer signals, and protected reporting.

---

# 3. Canonical product objects

## 3.1 Scope

The bounded part of the institution being governed.

Examples:

- a bank or legal entity;
- a country or jurisdiction;
- a banking channel;
- a critical business service;
- a product;
- a region or branch;
- a merchant population;
- a vendor-supported service;
- an asset or account population;
- a customer segment.

Scopes may be nested:

```text
Institution
└── Legal entity
    └── Country or region
        └── Channel or service
            └── Branch, merchant, process, system, vendor, or asset population
```

A regional bank may use a shallow hierarchy. A multinational bank may use deeper legal-entity, jurisdiction, product, and service boundaries. The operating model remains the same.

Every screen and decision must make the active scope visible enough to prevent wrong-entity or wrong-period action.

## 3.2 Exposure Pattern

A reusable description of how a banking activity, service, population, dependency, or control can fail or create harm.

Initial universal exposure families:

1. availability and resilience;
2. asset and inventory integrity;
3. identity and access;
4. transaction integrity;
5. reconciliation and settlement;
6. fraud and abuse;
7. data and privacy;
8. customer and conduct harm;
9. third-party and concentration dependency;
10. change and configuration integrity;
11. physical and environmental integrity;
12. regulatory, contractual, or policy non-conformance;
13. model or automated-decision failure;
14. evidence and data-quality uncertainty.

An exposure pattern is not an incident and not a permanent risk-register row. It is a reusable reasoning template that can be applied to ATM, POS, mobile, branch, card, payment, lending, treasury, vendor, cyber, and other contexts.

A pattern should identify:

- affected object types;
- causal conditions;
- possible consequences;
- common indicators;
- common claims;
- likely evidence sources;
- common controls;
- decision thresholds;
- and verification methods.

## 3.3 Risk Situation

A current, bounded instance of one or more exposure patterns requiring monitoring, evidence, decision, action, or verification.

Example:

> Thirty-one active ATM records in Lagos do not have a verified device, location, and branch relationship. Twelve have no recent switch heartbeat and seven locations have related tampering or card-retention complaints.

A risk situation includes:

- scope;
- applicable exposure patterns;
- source observations;
- affected services, customers, entities, assets, vendors, and obligations;
- materiality and appetite context;
- what changed;
- known facts;
- uncertainty and contradiction;
- required claims;
- required authority;
- current decision or action state;
- verification state;
- and history.

A situation may be:

- monitoring only;
- awaiting evidence;
- under assessment;
- awaiting decision;
- authorized for action;
- in progress;
- awaiting verification;
- verified effective;
- verified ineffective;
- indeterminate;
- superseded;
- or closed with accepted evidence.

A situation is the primary product object shown to most users. The underlying graph, risks, controls, evidence, incidents, obligations, decisions, and actions remain connected but should not force separate navigation.

## 3.4 Claim

A precise statement that can be supported, contradicted, qualified, or remain unresolved.

Examples:

- every active ATM is assigned to an approved location;
- every active POS terminal belongs to an approved merchant;
- settlement totals reconcile with switch transactions within tolerance;
- privileged access was reviewed for the complete population during July;
- payment failover meets the approved impact tolerance;
- the remediation prevented unauthorized account reactivation for 30 days.

A claim must have:

- subject and scope;
- effective period;
- purpose;
- materiality;
- evidence recipe;
- conclusion state;
- and version.

## 3.5 Evidence Recipe

A policy describing what observations are acceptable for a claim, from which sources, for which scope and period, and with what review.

Example:

```yaml
claim_type: asset_presence_and_assignment
subject_type: channel_device
required_facts:
  - device_identifier
  - assigned_location
  - responsible_owner
  - operational_state
acceptable_sources:
  inventory_database:
    authority: primary_for_assignment
  switch_telemetry:
    authority: primary_for_connectivity
  field_photo:
    authority: corroborating_for_visible_attributes
  branch_confirmation:
    authority: human_assertion
freshness:
  inventory_database: 24h
  switch_telemetry: 1h
  field_photo: 30d
minimum_policy:
  - inventory_database
  - switch_telemetry
  - one_of:
      - field_photo
      - independent_inspection
```

Recipes must distinguish:

- required facts;
- acceptable source types;
- source authority limits;
- coverage;
- freshness;
- independence;
- contradiction rules;
- approval requirements;
- and whether automated evaluation is permitted.

## 3.6 Observation

A normalized, source-preserving record of something seen, submitted, imported, measured, extracted, or asserted.

All capture methods produce observations with a consistent contract:

- subject;
- observed or asserted fact;
- value;
- source;
- capture method;
- effective time;
- capture time;
- scope;
- provenance;
- original artifact or source reference;
- extraction or transformation history;
- authority and limitation;
- confidence or review state;
- sensitivity;
- and version.

Observation sources include:

- API or event integrations;
- database and scheduled exports;
- spreadsheets and CSV files;
- forms and structured confirmations;
- controlled dropdown selections;
- photographs and document scans;
- screenshots;
- audio and video;
- email and messaging;
- staff attestations;
- vendor submissions;
- customer reports;
- protected reports;
- tests and inspections;
- and approved external intelligence.

An observation is not automatically a verified fact. Its authority depends on source ownership, scope, freshness, integrity, coverage, and the claim for which it is being used.

## 3.7 Conclusion

A versioned determination of what the current evidence supports.

Possible states:

- supported;
- partially supported;
- unsupported;
- contradicted;
- indeterminate;
- expired;
- or not applicable.

A conclusion identifies:

- included observations and evidence;
- excluded evidence and reason;
- contradiction state;
- assumptions;
- evidence sufficiency;
- evaluator;
- required approval;
- valid period;
- and supersession history.

## 3.8 Decision

An authorized selection among options in response to a situation, claim, incident, obligation, issue, or exposure.

A decision includes:

- context;
- evidence and conclusion;
- uncertainties;
- available options;
- expected effects and limitations;
- cost and dependencies where relevant;
- selected option;
- authority and segregation-of-duties checks;
- rationale;
- dissent or override;
- conditions;
- expiry or review triggers;
- action plan;
- and verification contract.

## 3.9 Verification Contract

A machine-readable definition of how ClearSight will determine whether the selected response achieved the intended observable outcome.

It includes:

- expected outcome;
- baseline;
- population or scope;
- measurement source;
- success and failure thresholds;
- observation period;
- required evidence;
- acceptance authority;
- and failure response.

The system verifies whether defined outcome criteria were met. It must not overstate that one action conclusively caused all observed risk movement.

---

# 4. Universal banking application

The model should be reused rather than replicated by domain.

## 4.1 ATM example

### Scope

Retail ATM channel, Lagos region, 428 machines.

### Exposure patterns

- asset and inventory integrity;
- availability and resilience;
- physical integrity;
- customer harm;
- vendor dependency.

### Observations

- 31 inventory records lack a confirmed device-location relationship;
- 12 ATMs have no heartbeat for more than 24 hours;
- 7 locations have related complaints;
- vendor service records disagree with internal inventory.

### Claims

- every active ATM is physically present at its assigned location;
- serial number matches the approved inventory;
- visible tamper protections appear intact;
- the machine communicates with the switch;
- responsible branch and vendor owners are known.

### Capture

- current inventory import;
- switch telemetry;
- targeted branch photo and structured confirmation for unresolved machines;
- vendor maintenance record where relevant.

The system asks about the unresolved population, not all 428 machines.

## 4.2 POS example

### Scope

Merchant POS channel, 18,000 active terminals.

### Exposure patterns

- asset and inventory integrity;
- merchant identity mismatch;
- transaction integrity;
- reconciliation and settlement;
- fraud and abuse;
- processor resilience.

### Observations

- terminal identifier appears from an unexpected location;
- settlement upload contains duplicate IDs;
- merchant KYC differs from the terminal-management record;
- reversal rate increased;
- processor availability declined.

### Claims

- every active terminal belongs to an approved merchant;
- terminal serial and logical identifier match;
- settlement totals reconcile with switch transactions;
- unusual location movement has been reviewed;
- processor availability remains within tolerance.

The same objects and workflow apply as for ATM. Only the channel pack, claims, evidence recipes, and thresholds differ.

---

# 5. Configuration model

ClearSight supports different bank sizes through configuration layers rather than custom product forks.

## 5.1 Base banking model

Universal concepts and exposure families.

## 5.2 Channel packs

Reusable packs for:

- ATM;
- POS and acquiring;
- mobile banking;
- internet banking;
- branch operations;
- agency banking;
- cards;
- payments and switching;
- lending;
- treasury;
- data and technology;
- vendor-supported services.

A channel pack may define:

- common scopes and entity types;
- exposure patterns;
- standard claims;
- evidence recipes;
- indicators;
- controls;
- capture templates;
- visual summaries;
- and verification patterns.

## 5.3 Jurisdiction packs

Define applicable obligations, reporting thresholds, retention, privacy, terminology, and authority requirements.

## 5.4 Institution profile

Defines:

- organizational and legal-entity hierarchy;
- terminology;
- critical services and channels;
- appetite and thresholds;
- approved sources and source authority;
- roles and authority;
- custom claims and recipes;
- and deployment or residency constraints.

---

# 6. Progressive integration model

ClearSight must not require perfect APIs before it becomes useful.

## Level 0 — Structured manual capture

- forms;
- mobile capture;
- photos and scans;
- spreadsheet upload;
- document upload;
- controlled dropdowns.

## Level 1 — Managed scheduled imports

- approved spreadsheets from managed locations;
- SFTP;
- database exports;
- controlled CSV feeds;
- recurring file ingestion.

## Level 2 — API synchronization

- IAM;
- HR;
- ITSM;
- CMDB and asset systems;
- vendor platforms;
- complaints and CRM;
- document systems.

## Level 3 — Event and telemetry integration

- switch events;
- service monitoring;
- identity changes;
- transaction or settlement anomalies;
- incident events;
- cloud and security telemetry.

Every level produces the same observation model. Banks can improve integration maturity without changing the product semantics.

---

# 7. Source Registry and data quality

Each source must have a governed Source Profile:

- source name and owner;
- system or collection method;
- authoritative fields and explicit limitations;
- scope;
- identifiers;
- expected freshness;
- current freshness;
- last successful import or synchronization;
- mapping version;
- known data-quality limitations;
- unresolved mappings;
- access and purpose policy;
- and health state.

Example:

```text
Source: ATM Asset Register
Owner: Head of Channels Operations
Authoritative for:
  - ATM serial number
  - assigned branch
  - owning vendor
Not authoritative for:
  - physical presence
  - live communication status
  - internal tamper condition
Freshness target: 24 hours
Current age: 18 hours
Unresolved mappings: 7
Known limitation: vendor IDs are not globally unique
```

Reconciliation states:

- matched;
- provisionally matched;
- unresolved;
- contradictory;
- rejected;
- or superseded.

Data-quality weaknesses are visible observations and may create or affect a risk situation. They must not be hidden behind a generic successful-integration badge.

---

# 8. AI role

AI acts as a governed compiler between messy institutional inputs and structured product objects.

```text
Photo, spreadsheet, document, email, narrative, or voice
→ extract and normalize
→ propose observations, entities, relationships, claims, and questions
→ validate against schema, source policy, and authorization
→ request human confirmation where required
→ persist governed structured output with lineage
```

AI may:

- extract;
- transcribe;
- classify;
- normalize;
- compare;
- suggest mappings;
- identify possible contradiction;
- draft focused requests;
- summarize;
- and explain.

AI does not determine authority, silently approve material conclusions, alter appetite, close major findings, or replace source evidence.

---

# 9. User experience boundary

Most users operate through five surfaces.

## Today

A role-specific brief of material situations, evidence gaps, decisions, failed verification, and important deadlines.

## Situation

One workspace combining summary, evidence, decision, action, outcome, and history for a bounded situation.

## Capture

A lightweight surface for one focused request, photo, scan, spreadsheet, structured confirmation, correction, or discrepancy.

## Explore

An analyst surface for scopes, populations, relationships, sources, exposure patterns, situations, claims, observations, obligations, controls, incidents, decisions, and outcomes.

## Configure

A restricted surface for source registry, channel and jurisdiction packs, controlled vocabularies, evidence recipes, appetite, authority, access, retention, and automation.

The graph, evidence engine, decision ledger, workflow runtime, and AI operator platform are architectural capabilities. They must not become mandatory navigation concepts for ordinary users.

---

# 10. Product invariants

1. Situations before modules.
2. Banking language before GRC jargon.
3. Scope is always clear before action.
4. Existing evidence before human requests.
5. Source authority before automated trust.
6. Observations retain original provenance.
7. Data-quality weakness remains visible.
8. Evidence before confidence.
9. Contradiction before false certainty.
10. Decisions before dashboards.
11. Verification before closure.
12. Human authority for material judgment.
13. Progressive integration over perfect-integration dependency.
14. Progressive disclosure over interface density.
15. Institutional memory over periodic snapshots.
16. Internal architecture must not become user-interface architecture.

---

# 11. Definition of success

The operating model succeeds when:

- a regional bank can begin with spreadsheets, forms, and mobile capture;
- a national or multinational bank can add APIs, events, legal entities, jurisdictions, and deeper authority without changing product semantics;
- ATM, POS, mobile, branch, payments, cyber, vendor, and other risks reuse the same exposure, situation, evidence, decision, and verification logic;
- users see familiar banking situations rather than module boundaries;
- staff are asked only for missing facts;
- data-source weaknesses remain explicit;
- executives see fewer but more useful items;
- and every material situation can be reconstructed from original observations through conclusion, decision, action, and verified outcome.