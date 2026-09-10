# Employee work profile and activity design

**Date:** 2026-09-10
**Status:** Approved by the operator on 10 September 2026

## Outcome

ClearSight will give an internal employee a durable, addressable work profile. A person named on an issue, action, Program, oversight row, activity row or People inventory can open the profile when the viewer has the required scope. The profile makes current responsibility, completed work and recorded system activity inspectable without turning identity administration into a second operational workspace or producing an employee score.

The initial seeded profile must make Hakeem's source-backed Cloudspace remediation work understandable. Hakeem is an internal Action performer, not a third-party response field or a vendor identity. His profile must show the five imported Cloudspace Actions, their March 31, 2026 deadlines and their current state from the canonical Matter/Action records. It must not imply that he performed an action merely because somebody assigned work to him.

## Evidence and current gap

The demo employee manifest and demo authenticator already create Hakeem as a real internal `PERSON` with the supplied non-production sign-in `hakeem@demo.com` / `password`. His position is **Sample POS Business action owner** in the **POS Business** function.

The existing application has:

- source-backed principals, positions and role bindings;
- current Action and Matter ownership containing durable principal IDs;
- a legal-entity-scoped Oversight projection with owner workload/history context;
- an actor-filterable, keyset-paginated System Activity read model over canonical outbox, governance-decision and recovery events.

It has no person route. Several stored-responsibility display DTOs intentionally provide only a display name, so the browser cannot navigate from a historical attribution safely. The profile must add a read-side reference; it must not cause the browser to submit a principal identifier as authority for a command.

## Approaches considered

### 1. Dedicated employee work profile — selected

Add a `#/people/{principal-id}` route, governed profile APIs and a reusable internal-person link. The profile has separate Work and Activity sections, direct deep links to the relevant record, and explicit freshness/coverage.

This is selected because it is reusable from the Action card that prompted the request, Oversight, Configure and System Activity. It preserves a durable URL and lets desktop and mobile use a full operating surface rather than an overloaded popover.

### 2. Profile flyout from each name

Show a compact pane with assignments and recent events. This is faster to add but cannot comfortably support pagination, audit filtering, a narrow-screen reading order or a durable audit reference. It repeats the shallow flyout treatment already rejected for vendor work.

### 3. Redirect names into Configure → People

Reuse the existing directory inventory and System Activity filters. This mixes identity administration with operational work, is restricted to configuration readers, and does not preserve the source Action or Program context. It is rejected.

## User experience

### Entry points and route

`#/people/{principal-id}` is the canonical employee work-profile route. Browser Back returns to the originating record through normal history.

The shared `PrincipalLink` renders an internal, resolvable `PERSON` name as an accessible link only when the signed-in actor may open a profile. It is used in:

- Action owner and accountable-owner facts;
- Program owners, safeguard performers and evidence reviewers;
- Oversight workload/history rows and intervention owner names;
- System Activity actor names;
- Configure → People inventory.

External participants, vendors, services, teams, queues, committees and unavailable historical labels stay as non-interactive text. A historical principal reference may remain clickable only when the server can still resolve that principal in the viewer's tenant and legal entity.

### Profile layout

The page is a dedicated operating record, using the established premium institutional surface language and shared component contracts.

The header contains a monogram, the employee name, active/inactive status, current position and function. It identifies sample records as **Sample data**. It does not expose a sign-in address, session information or directory-source detail to viewers without identity-read permission.

The first viewport contains four current-work measures: active responsibilities, overdue responsibilities, blocked responsibilities and work awaiting outcome confirmation. Every measure identifies its exact visible population and `as_of` time. An unavailable or incomplete calculation displays **Unknown** with the recovery state; it never substitutes zero.

Two peer sections follow:

1. **Assigned work** lists current responsibilities first, then historical responsibility intervals when recorded. Each row identifies the responsibility, record, current state, deadline, assignment/reassignment source and a direct record link. Assignment records name the initiating actor separately from the recipient.
2. **Activity** is a keyset-paginated chronological log of actions whose verified actor is the employee. Rows show time, action, target record, source and exact record link where the viewer may read it. Filters are limited to date range and activity category. A separate **Assignment history** group exposes assignment/reassignment events that concern the employee but were performed by another actor.

For Hakeem, an initial activity-empty state is valid when no event names him as actor. The Assignment history must still show that the relevant Cloudspace actions were assigned to Hakeem by their recorded initiator. The profile never fabricates an action, completion, review or outcome.

At desktop widths, the header/metrics and work rows use a dense two-column composition. At tablet widths, metrics become two columns. At mobile widths, metrics and row facts stack in source order; Work and Activity use the existing compact Tabs replacement. Links and action targets keep a 44px hit area.

## Read model and API boundary

The profile is a read model over existing authoritative records. It creates no employee-performance table and stores no duplicate event payload.

`GET /api/v1/people/{principal-id}/work-profile` returns a bounded initial view:

- resolved person summary and current position/function when allowed;
- current-work metrics and their as-of/freshness/coverage metadata;
- current assignments and a bounded first page of assignment history;
- capability flags indicating whether Activity and identity detail may be shown.

`GET /api/v1/people/{principal-id}/activity` accepts a bounded page size, keyset cursor, category and date range. It returns only events for which the person is the verified actor. Assignment targeting is not folded into this feed.

The activity endpoint reuses the existing normalized activity sources, exact actor filtering and keyset semantics. It additionally receives the verified legal-entity scope and resolves object visibility before returning a row or record link. It does not turn the unrestricted operational activity reader into an employee directory.

Current and historical work comes from canonical Matter, Action, Workflow Task, Program and evidence-assignment records. The API computes only a bounded legal-entity and authorization-filtered population. Each row retains its canonical record identifier; the browser does not calculate responsibility, infer owners or assemble audit facts across broad lists.

Stored responsibility display DTOs gain a read-only `principal_id` only when it is a resolved internal-person reference. Command handlers continue to ignore browser-supplied actor, assignee, reviewer and approver IDs and re-evaluate current authority at execution.

## Access and privacy

The profile route is available to:

- the verified employee for their own profile;
- an actor with `OVERSIGHT_READ` or `IDENTITY_READ` in the same tenant and legal entity.

The browser renders `PrincipalLink` only when the profile capability is returned in actor context. A direct request outside these conditions returns not found without revealing whether the principal exists.

Identity-read governs directory-source detail and access/role detail. Oversight readers receive the operational profile, but not credentials, sign-in metadata, group membership or material authority configuration.

All work and activity rows are filtered server-side before limits. Restricted Matters and linked protected records use the existing fail-closed visibility policy. An inaccessible target is omitted rather than replaced with a revealing title. The response exposes an explicit incomplete/unknown coverage state when a source required for a count is unavailable; it does not report an inferred total.

Raw system/configuration audit events and governed exports retain their existing higher platform/audit permissions. The profile's Activity feed contains only the safe, visible employee activity contract above.

## Error and recovery states

- A missing, cross-tenant, cross-entity or unauthorized person returns the standard not-found state.
- An inactive former employee retains visible historical attribution when authorized; current work measures state that no active position is held.
- A directory/position lookup failure preserves readable work rows but marks identity detail unavailable.
- A work or activity source failure preserves the successful section, marks the failed population unknown and supplies a retry action.
- No current work, no historical assignment and no personal activity each name the checked population and do not imply complete system history.
- Pagination, date/category filtering and record opening remain server-authorized. A formerly visible record that is no longer accessible opens its ordinary not-found state.

## Seed and source integrity

No new artificial profile records are seeded. The existing employee manifest remains the identity source for Hakeem, Blessing and the other sample employees. The profile derives Hakeem's current work from the source-backed Matter and Action IDs produced by the third-party register migration.

The response treats Blessing/Joel as internal assessors, Hakeem as Action performer, POS Business as accountable function and Cloudspace as the vendor relationship. None becomes a vendor response answer solely to populate a profile.

## Verification

This change avoids a new broad seed-regression suite. Focused checks cover:

- actor/self, oversight-reader, identity-reader, cross-tenant, cross-entity and unauthorized profile access;
- no client-supplied authority in profile or follow-on commands;
- server-side restricted-record filtering before pagination;
- separate actor Activity versus assignment-recipient history;
- Hakeem's five Cloudspace Actions, source deadline and truthful no-personal-activity state when applicable;
- internal person links, non-interactive external/unavailable labels and browser Back/deep-link recovery;
- unavailable, empty, inactive and paginated states;
- desktop and mobile rendered proof in both themes, plus focused accessibility/copy checks.

The implementation must update the current execution ledger, relevant API contract and design evidence before deployment.
