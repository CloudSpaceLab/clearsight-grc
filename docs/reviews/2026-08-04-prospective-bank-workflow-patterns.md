# Prospective Bank Workflow Pattern Review

**Review date:** 2026-08-04  
**Source:** privately supplied prospective-bank workflow samples.  
**Publication rule:** this review intentionally excludes the institution’s name, staff names, branches, vendors, exact figures and raw records because the ClearSight repository is public.

---

# 1. Executive finding

The samples confirm that the practical GRC problem is not the absence of registers. The institution already maintains many registers, checklists, workplans and reporting workbooks.

The problem is that the same institutional reality is repeatedly represented in separate spreadsheets and manually reconciled through email, comments, phone calls and dashboard preparation.

The sample estate included patterns corresponding to:

- compliance obligations and deadlines;
- IT asset inventory and IT risk assessment;
- risk exceptions and remediation comments;
- annual IT risk and assurance workplans;
- third-party risk findings;
- privacy and data-protection checklist obligations;
- branch and head-office KRIs;
- operational loss data;
- business impact analysis and dependencies;
- RCSA;
- IT governance dashboard and integration requirements.

The winning ClearSight design is therefore not a more attractive digital register. It is a shared operating layer that allows these views to derive from the same authoritative sources, relationships, cases, situations, decisions and evidence.

---

# 2. What the samples reveal

## 2.1 Compliance registers are working summaries, not primary law

The compliance-register pattern contains regulator, requirement summary, deadline or frequency, evidence, rating and status fields.

Important limitations:

- source document and exact provision may be absent;
- multiple kinds of obligation are mixed together;
- recurring requirements and one-time circular changes share the same row model;
- rows may be manually added without validation;
- evidence, status and ownership are often incomplete;
- amendment and supersession are difficult to represent;
- one row cannot express multiple legal entities, systems, controls or implementation paths.

ClearSight implication:

- import rows as secondary observations;
- require source reconciliation before approved obligation status;
- preserve source provision, interpretation, applicability and effective period separately;
- derive register views from governed obligations.

## 2.2 Structured privacy checklists are closer to the desired model

The privacy-checklist pattern includes:

- control area;
- requirement;
- legal reference;
- applicability;
- timeline or frequency;
- evidence required.

This is valuable seed material for Regulatory Obligations and Evidence Recipes.

It still requires:

- source-version validation;
- legal and applicability approval;
- mapping to policies, processes, systems, vendors and owners;
- implementation state;
- evidence versions and sufficiency;
- amendment handling;
- independent assurance conclusions.

## 2.3 Risk-register spreadsheets collapse many different concepts

The IT risk-register pattern combines:

- asset inventory;
- threat and vulnerability;
- confidentiality, integrity and availability impact;
- likelihood and risk matrices;
- existing controls;
- inherent risk;
- treatment option;
- proposed controls;
- standards references;
- target dates;
- residual risk;
- status, comments and responsibility.

ClearSight implication:

These should not remain one wide record. They map to shared Asset, Exposure Pattern, Risk Situation, Control Implementation, Assessment, Decision, Action and Verification objects.

## 2.4 Exception handling is currently comment-driven

The exception-register pattern contains detailed findings, affected applications, implications, recommendations, framework references, target dates, status and long free-text owner/risk-team comments.

The comments reveal manual routing and responsibility ambiguity.

ClearSight implication:

- ownership and authority must be structured;
- redirect, delegation, challenge and rejection need explicit states;
- comments remain immutable communication observations;
- assignment changes and decisions must not be inferred from prose alone;
- action completion must remain separate from evidence acceptance and verified outcome.

## 2.5 Workplans are assurance schedules, not risk records

The annual-workplan pattern schedules reviews across applications, technical risk areas, recurring security activities and vendors.

ClearSight implication:

- represent planned Review Activities with scope, owner, frequency and evidence expectations;
- link each activity to assets, vendors, controls, obligations and exposure patterns;
- create findings or observations only from completed review results;
- avoid duplicating risk or vendor records inside the plan.

## 2.6 Third-party risk is finding-centric and document-heavy

The vendor-register pattern records service provider, service, findings, implications, rating, recommendation, owner, timeline, status and vendor/business comments.

Common evidence includes certifications, testing reports and contract clauses.

ClearSight implication:

- separate vendor entity, service relationship, assessment, finding, evidence, contract obligation and remediation;
- treat vendor attestation differently from independent evidence;
- track certificate effective periods and stale evidence;
- link the same vendor and evidence to multiple bank services without duplicate upload;
- verify remediation rather than accepting a commitment comment.

## 2.7 Branch KRIs are very large recurring questionnaires

The branch-KRI pattern asks many monthly questions covering cash operations, physical security, equipment, fire safety, power, environmental conditions, customer data, incidents and workplace safety.

ClearSight implication:

- define each indicator and its source hierarchy;
- prefill data from loss, incident, asset, maintenance and monitoring systems;
- ask branches only for facts that remain unresolved;
- dynamically hide non-applicable questions;
- use conditional follow-up instead of one wide form;
- preserve denominator, period, branch and respondent authority;
- create situations from threshold breaches or contradictions rather than producing only a monthly spreadsheet.

## 2.8 Head-office KRIs already contain enforcement-work signals

The head-office KRI pattern includes counts of requests from law-enforcement and intelligence bodies, including differentiation by legal-instrument state, alongside customer complaints, channel failures, legal orders, ATM availability, security incidents and other metrics.

ClearSight implication:

- authority requests must exist as governed cases beneath the KRI;
- case type, legal review, deadline, subject count, directives and response status should drive the metric;
- aggregate views must not expose protected case subjects;
- recurring requests may reveal systemic KYC, records or monitoring weaknesses but must not imply subject guilt.

## 2.9 BIA is the strongest existing relationship dataset

The BIA pattern captures:

- business functions and processes;
- customer, financial, staff, legal, dependency and reputation impacts;
- expected outcomes;
- applications and resources;
- upstream and downstream business units;
- external dependencies;
- vital records;
- minimum staffing;
- maximum outage, RTO and RPO.

ClearSight implication:

This can seed the institutional context needed to interpret regulatory change, operational incidents, vendor exposure and resilience requirements. It should not remain isolated from the service, asset, vendor and risk model.

## 2.10 Loss data is disconnected from incidents, controls and recoveries

The loss-database pattern records loss type, amount, transaction description, branch hierarchy, occurrence and recognition dates, root cause, recovery and currency.

ClearSight implication:

- loss, recovery, incident, disciplinary/legal action, finding and remediation need explicit relationships;
- repeated root causes should update exposure patterns and control conclusions;
- regulatory sanctions and legal losses can link back to missed obligations or authority cases;
- monthly duplication or copied entries must be detectable.

## 2.11 Dashboard requirements describe a reporting layer over fragmented sources

The dashboard requirement pattern asks for a single source of truth across projects, ITSM, change, assets, budget, disaster recovery, channel performance and staffing, with data from several enterprise tools.

ClearSight implication:

The dashboard must be derived from governed domain objects and source health. AI summaries and predictive scoring cannot repair inconsistent identifiers or missing relationships. Source Registry, Observation normalization and shared identifiers must precede executive analytics.

---

# 3. The external-authority lifecycle missing from the spreadsheets

The samples show regulatory obligations and aggregated authority-request KRIs, but not one connected lifecycle for:

```text
Authority source
→ source authenticity and legal status
→ exact provisions or case directives
→ applicability or legal review
→ affected entities or case subjects
→ controls, case tasks and owners
→ evidence and decisions
→ authority response or implementation
→ acknowledgement and verified outcome
→ KRI, risk, incident, loss and committee views
```

This gap is addressed by [`../product/regulatory-and-enforcement-intelligence.md`](../product/regulatory-and-enforcement-intelligence.md).

---

# 4. Recommended migration order

1. Import institution, legal entities, branches, business units, channels, applications and vendors.
2. Create Source Profiles for each spreadsheet and enterprise source.
3. Import BIA and asset relationships as provisional institutional context.
4. Import compliance rows as source-unverified candidate obligations.
5. Import privacy checklist rows as candidate obligations and Evidence Recipes.
6. Import risk, exception and vendor findings as distinct situations, findings, actions and communication observations.
7. Import workplans as Review Activities.
8. Import KRI definitions, thresholds and historical observations.
9. Import incidents, losses and recoveries with duplicate detection.
10. Introduce Authority Sources, Regulatory Change Situations and Authority Request Cases.
11. Gradually derive future register and KRI views from governed source records.

---

# 5. Product conclusion

The customer samples validate the simplified ClearSight thesis:

> **Users should operate bounded situations and cases, while registers, workplans, KRIs and dashboards become purpose-specific views over one governed institutional record.**

The highest-value early workflows are:

- authoritative regulatory document to approved obligation and control programme;
- external-authority request to protected case, KYC/address/records workflow and response package;
- risk finding or exception to assigned action and verified remediation;
- branch KRI collection that pre-fills system evidence and asks only missing facts;
- BIA and source data reused as institutional context rather than re-entered in each module.