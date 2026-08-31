# Enterprise shell and Configuration decision brief

**Issue:** #108  
**Baseline:** `main@7bfec07a6b33d7399dab5d015ccfb1c0d09c3b21`

## Product job and primary users

ClearSight is repeated-use bank operating software. The shell must prioritize the work that bank users perform repeatedly while keeping restricted administration, demo tooling and ingestion mechanics available without presenting them as equal product modules.

Primary shell users:

- executives and control-function leaders acting through Today;
- Program owners maintaining continuing compliance;
- reviewers and operators handling issues, changes and evidence in Work;
- third-party risk users operating Vendors;
- form owners and reviewers operating governed Forms.

Primary Configuration users:

- identity/access administrators;
- governance/routing administrators;
- source/integration administrators;
- AI/automation administrators;
- platform operators.

Ordinary users should rarely enter Configuration.

## Baseline problem

The baseline shell mixed different capability classes in one navigation array:

`Today · Programs · Forms · Vendors · Work · Imports · Explore · Configure`

In demo presentation, `Explore` was a primary destination even though its executable content was explicitly sample/reference journeys. `Imports` was treated as a permanent module even though exact import work is normally entered from an intervention or originating workflow.

The baseline Configuration surface also loaded and displayed routing checks, workflow ownership, projection operations, governance administration, AI governance, automation policy and identity/access administration together. Opening Configuration triggered all top-level domain requests before the administrator selected a job.

`AutomationPolicies` additionally rendered `IdentityAccessPanel`, coupling unrelated administrative domains by component composition.

## Alternatives considered

### A. Restyle the existing Configure page

Rejected. Better spacing/cards would leave the information architecture, eager loading and domain coupling intact.

### B. Keep every domain as a global navigation module

Rejected. It would increase shell burden and make administration/demo/ingestion look equivalent to repeated operating work.

### C. Enterprise operating shell + restricted control plane

Selected.

```text
Operating workspaces
Today · Programs · Work · Vendors · Forms

Utilities
Search · Notifications · Help (when executable)

Administration
Configuration

Demo tooling
Demo environment · Reference journeys · role switching
```

Exact import deep links remain valid. Import administration moves under Data & integrations; operational import review continues to launch directly from the relevant work item.

## Selected Configuration structure

```text
Configuration
├─ Overview
├─ People & access
├─ Authority & routing
├─ Data & integrations
├─ Automation & AI
└─ System operations
```

The structure is based on administrator jobs rather than package boundaries.

### Overview

A quiet administrative index. It must not become a KPI wall or invent health percentages. Each row explains the administrative area and opens it.

### People & access

Identity sources, directory-backed people/groups and role mappings. Directory eligibility remains distinct from material decision authority.

### Authority & routing

Routing integrity, governance policies, delegation, escalation and supporting workflow-ownership context. Canonical assigned work remains in Today/Work.

### Data & integrations

Document imports and source/connection/mapping administration as existing executable capabilities become productized. No integration-marketplace theatre.

### Automation & AI

Approved automation policies and governed AI workload/policy state. AI-specific human approvals remain canonical Matter/Today work.

### System operations

Projection/background processing health and operator recovery. Operational lag must not visually masquerade as a compliance or policy failure.

## Interaction grammar

Every Configuration domain should converge on:

`Current state → inspect → explicit create/revise → preview/impact where material → submit/maker-checker → receipt/history`

Do not render large create/edit forms by default when the administrator is only inspecting current state.

## Loading and performance decision

`App.tsx` owns shell/runtime/navigation state, not every Configuration inventory.

- Configuration is lazy-loaded.
- The overview does not fan out across every admin API merely to look analytical.
- Each selected domain owns its bounded data load and retry state.
- Identity/access no longer mounts with automation policy content.
- Projection operations load only in System operations.

This reduces initial Configuration request fan-out and narrows failure domains.

## Demo posture

Enterprise presentation is the normal product presentation. Explicit demo presentation is supporting tooling.

- normal resolver result: enterprise;
- `?demo=1`: explicit demo presentation;
- static demo build: explicit demo presentation;
- server `demo_mode` describes backing environment and does not itself redefine product information architecture.

The legacy `#explore` route is retained temporarily only as a demo-backed Reference journeys deep link.

## Required states

Each Configuration domain must support:

- loading;
- live/default;
- empty named scope;
- partial/degraded;
- unavailable/retry;
- read-only/restricted where applicable;
- optimistic conflict in governed mutation flows;
- success/receipt;
- long content;
- keyboard/focus;
- 200% zoom/reflow;
- reduced motion.

A failure in one domain must not visually collapse unrelated domains.

## Responsive replacement

Desktop may use a narrow internal Configuration rail beside one working surface.

Tablet collapses simultaneous context while preserving the active domain and primary action.

Mobile keeps daily operating work in the bottom navigation. Configuration is reached through the administrative shell affordance and its internal navigation becomes a horizontally scrollable area selector. Complex focused configuration actions should become full-screen flows rather than squeezed desktop forms.

## Visual posture

- dense institutional rows over a mosaic of large cards;
- no populated-state hero illustration;
- restrained transparency only where it improves hierarchy;
- semantic state close to the object it describes;
- one dominant administrative action per state;
- no invented scores or decorative charts;
- Configuration typography smaller and denser than marketing/onboarding surfaces.

## Acceptance evidence

Before this redesign can merge:

1. TypeScript typecheck passes on exact head.
2. Existing rendered state/accessibility tests are migrated to the enterprise-first navigation contract and pass.
3. Production build passes.
4. Deterministic Chromium review passes with the new shell.
5. Configuration overview and at least Authority/System operations are inspected at desktop, tablet and mobile widths.
6. Direct import and legacy demo-reference deep links remain safe.
7. No backend/durable configuration model or new global state framework is introduced.
