# V2.12 — internal roles and responsibilities

## Outcome

The internal interface now starts from a person's real responsibilities and current scoped authority. **My workspace** presents assigned actions, the programs the person actually owns, and relevant next steps. A person with several responsibilities can switch between capture, ownership, independent review, leadership reporting and access administration. The selected lens changes presentation only; it never writes a grant or overrides an explicit deny.

The implementation uses the existing `S.accessControl`, named program and assessment records, work projections and canonical commands. No parallel identity, authorization or work-item store was added. `S.roleExperience` holds only the preferred lens per person.

## Findings addressed

1. The former sidebar and search starting points exposed the same destinations to every working identity, even where no current grant covered the selected location.
2. A contributor, independent reviewer and access administrator had no distinct starting point. Work was available, but the next relevant action was not organized around their recorded responsibility.
3. Several older control, RCSA, branch KRI and privacy commands accepted a selected preparer or reviewer name without matching the working identity. Hiding their buttons would not have corrected those commands.
4. Some source editors remained editable after identity or grant changes, although the source was only readable to the new actor.
5. The adoption flow's organizational scope pin lived in an old action wrapper. A later wrapper replaced that action, so the canonical creation path could create a program with no exact organizational scope. The parent integration moved this pin into `app.js`'s canonical `publishReview` program object.

## Internal experience

### My workspace

Route: `my-workspace`.

- Shows the working identity, selected organization location and active role assignments.
- Derives available responsibility lenses from current scoped capabilities and verified program/response ownership.
- Presents the acting person's open work in deadline order. Review and capture lenses use the appropriate subset of that existing queue.
- Keeps source actions exact: a task opens its action detail, a response its assignment, a review its submitted source.
- Retains unavailable assigned work with an access explanation, rather than presenting a working action that cannot be completed.
- Shows owned programs with named response progress and a direct assignment-management link.
- Offers relevant routes to Documents, branch returns, supplier onboarding, independent review, reporting or access administration.
- Keeps a combined view for people with several responsibilities.

### My access

Route: `my-access`.

- Shows a module/action matrix for the selected location.
- Distinguishes complete coverage, partial coverage and unavailable actions.
- Inspects one exact permission and location using canonical grant/deny traces.
- Lists active assignments, review deadlines, expiry dates and direct restrictions.
- Explains that named program and response responsibility does not grant unrelated editing, document export or access administration.

### Navigation, search and read-only state

- The sidebar reflects available module-view coverage and legitimate named program work. Personal Work and Inbox stay accessible.
- The brand returns to My workspace; the separate scope dashboard remains owned by the reporting module.
- Search filters both fresh and cached results when identity, scope or permissions change. Record results preserve their exact IDs.
- A known inaccessible destination gives a scoped access explanation and a route back to My workspace.
- Visible write actions and source forms use the same capability or named-authority checks as their corresponding commits. Read-only source inspection remains available where the person can view the source.
- Review forms record the actual working reviewer. Selecting a person's name cannot impersonate that person.
- The interface uses the existing visual tokens, focus styles, responsive sidebar and modal lifecycle. It adds no second main landmark.

## Public API

`window.CLEAR_SIGHT_ROLE_EXPERIENCE.version === '2.12'`

| Method | Contract |
| --- | --- |
| `identity()` | Returns the actual local working person's `{id, name}`. |
| `capability(permission, options)` | Evaluates the canonical policy at every exact scope. Options accept `scopeId`, `scopeIds`, a typed `subject`, optional preview `userId`, and source `makerId`. Returns `allowed`, `partial`, `reason`, `scopeIds`, `checks`, and flattened `trace`. |
| `coverage(permission, options)` | Evaluates the selected node and every descendant, or all actual organization nodes for the whole-organization view. Returns `any`, `full`, `allowedScopeIds`, `deniedScopeIds`, `roots`, `checks`, and `reason`. |
| `require(permission, options)` | Checks a command and throws on refusal. It always forces the actual `recActor()` identity, even if a caller supplies another `userId`. |
| `canViewSubject(type, id)` | Resolves the canonical record, navigation boundary and exact view authority. Returns a decision object, not a boolean. |
| `context()` | Returns current identity, scope, coverage, active assignments, verified ownership and available responsibility lenses. |

### Scope semantics

`capability('documents.edit', {scopeId: ''})` means exact whole-organization authority. It does not mean that one branch grant is sufficient.

Supplying several exact scopes requires **all** of them. An empty, missing or invalid subject boundary fails closed. An explicitly empty or unknown preview user fails closed. Invalid stored navigation scope does not widen the permission decision.

Coverage evaluates descendants separately so a child-level deny cannot be concealed by an allowed ancestor. `roots` contains only complete permitted subtrees; it never labels a subtree containing a denied child as fully available. Navigation can use `any`; complete aggregate/report decisions use `full` and their captured broader-source boundaries.

The role module does not modify `selectedScope`, the scope selector or report population. `scoped-analytics.js` owns those contracts. Published report subjects delegate to that module's `snapshotAccess`, so current navigation does not replace the captured pack boundary.

### Shared-document read rule

An explicitly shared reference (`scopeMode === 'shared'`) is readable when `documents.view` has some permitted coverage within the selected location. At whole-organization navigation, this means at least one actual assigned location is readable. Explicit denies still apply to each location that contributes coverage.

This shared-reference rule applies **only to viewing**. Shared document mutations and exports require their exact whole-organization permission. Scoped files use their exact `orgScopeId`. A scoped document without a valid location does not become shared by default. Classification remains metadata; this module does not create an ownership-only rule for Restricted files. The module does not call or wrap `docVisible`, avoiding recursion with the document-management integration.

Suitable document permission IDs are `documents.view`, `documents.edit`, `documents.submit`, `documents.approve`, `documents.export`, and `documents.manage`.

## Consequential command changes

| Workflow | Additional command boundary |
| --- | --- |
| Control implementation | Current and proposed locations must both permit `risks.edit`; original source-digest checks remain. |
| Control tests | The acting person must be the named tester when submitting. The assigned reviewer must be the actual acting reviewer and hold exact independent approval authority. |
| RCSA | Draft and submission use the recorded first-line owner. Decisions use the assigned actual reviewer and exact risk scope. Effective rating, source drift and document-version checks remain unchanged. |
| Branch KRI | The actual return owner prepares/submits the return; the actual assigned cycle reviewer decides. Modal confirmation rechecks identity, location and permissions. |
| Continuous indicators | Observation preparation and commit require the indicator's exact KRI-edit authority. Dated observation history remains canonical. |
| Privacy | Preparation, submission and decisions use actual ownership/reviewer identity and exact scoped authority. Legacy cases lacking named-reviewer metadata remain reviewable by an independent actor with current scope authority. |
| Program adoption | Exact selected or target scope is checked at preparation and confirmation. Canonical program creation now pins that scope directly, before named-assignment initialization. |
| Shared form definitions | Creating, changing or publishing the shared catalogue requires exact whole-organization `programs.edit`. Named target responses retain their separate assigned-person authority. |
| Named assessment work | Verified program ownership and respondent/coordinator assignments remain legitimate source authority. Exact explicit edit/submit restrictions override them. Public `CLEAR_SIGHT_ASSESSMENTS.assign` points to the checked command. |
| Ordinary actions | Other people can inspect a task. Only its actual recorded owner can complete or reopen it; canonical typed source scopes, source availability and explicit restrictions still apply. Source approvals remain separate from action completion. |
| Access and escalation | Existing guarded canonical save/apply commands remain. Real action IDs and editable draft fields now reflect the required management authority; access exports cover their complete organization-wide payload. |

Existing vendor owner authority, document source checks, loss financial commands, assignment source snapshots and versioned independent decisions are preserved. The module never changes a live grant when the user changes a responsibility lens.

## Integration and files

Load `role-experience.js` after all V2.11 interaction modules and before `scoped-analytics.js`, `document-management.js`, and `external-assessments.js`. The module registers routes and wraps functions; it does not bootstrap or render on load.

New owned files:

- `assets/role-experience.js`
- `assets/role-experience.css`
- `docs/ROLE_EXPERIENCE_V212.md`
- `tests/role_experience_v212_checks.js`
- `tests/role-experience-v212-results.json`

The parent owns static route entries, build ordering, the runtime harness and the canonical adoption fix in `assets/app.js`.

## Validation

The targeted suite passes **20/20 actual-source checks**. It covers exact permission scope, active dates, expired assignments, explicit deny precedence, child-deny coverage, maker independence, shared-document read/mutate distinction, missing scopes, role-title non-entitlement, lens non-grant behavior, cache-sensitive search, reporting-only navigation, named owner/response authority, exact command identity, revoked grants after preparation, correction and review transitions, legacy case review, task inspection/completion and registered/escaped route markup.

Relevant integrated regressions passed:

| Suite | Passing checks |
| --- | ---: |
| KRI V2.8 model and rendering | 11 |
| Operational workflows V2.9 | 14 |
| Cross-workflow continuity V2.9 | 25 |
| Shared interactions V2.11 | 15 |
| Assurance interactions V2.11 | 17 |
| Operational interactions V2.11 | 16 |
| Governance interactions V2.11 | 19 |
| Privacy interactions V2.11 | 10 |
| Vendor onboarding V2.10 | 15 |

The six authorized legacy test changes preserve the original source, financial, workflow and snapshot assertions. Originals are retained under `tests/history/*.pre_v212.js`:

- Enugu KRI rendering now uses Daniel's real branch identity.
- Independent decisions now switch to Ifeoma before calling the command and back to the first-line person for corrections.
- Head Office preparation tests give Amina only a synthetic `U-HO-SEC` contributor assignment, keeping contributor approval denies. RCSA tests name her as the explicit first-line cycle owner.
- Search and the unrelated-person assignment queue use existing global readers, rather than an actor with no access to that source.
- Board publication uses Chidi's existing reporting view/export authority and explicit synthetic `reports.edit`/`reports.submit` grants for the captured whole-organization fixture. No seed grants are changed.
- The adoption regression additionally asserts the new program's exact `U-IK-PAY` scope.
- Action-completion tests begin as the actual task owner, change identity during confirmation to verify rejection, then return to the owner. The typed-source check removes the source after opening the completion dialog, verifies that commit is rejected, and asserts that the task remains open.

Tests use the real application modules with a substituted DOM/browser shell. Browser layout, focus behavior in a real browser, authenticated retrieval and server enforcement are not claimed. The package remains a local synthetic design reference; production identity and authorization must use the same domain boundaries on the server.
