# Explicit Overdue Work, Production Copy and Imports Theme Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make passed Action deadlines explicit, replace implementation narration across the affected Work/Program/Forms journeys, and give the complete Imports workflow readable semantic surfaces in light and dark themes.

**Architecture:** Add one deterministic deadline presentation function beside the existing date conversion functions and render its derived state without changing the Action lifecycle. Treat customer copy as executable product behavior by extending the existing source scanner and rewriting every resulting affected-workflow violation. Register the Imports stylesheet with the existing UI contract and replace local dark values with the current semantic token layers.

**Tech Stack:** React 19, TypeScript 7, Vitest, Testing Library, axe-core, CSS custom properties, Node test runner, Playwright, Vite.

---

## File structure

- `web/src/dueDate.ts` owns selected-date conversion and the new deterministic Action-deadline presentation.
- `web/src/dueDate.test.ts` proves calendar, invalid, terminal and overdue semantics independently of React.
- `web/src/components/MatterActionsPanel.tsx` renders lifecycle and deadline as separate visible states.
- `web/src/matter-record.css` owns the Action-card layout and overdue emphasis.
- `web/src/components/MatterRecordWorkspace.test.tsx` proves the complete record displays overdue text and both status labels.
- `web/src/copyQuality.test.ts` rejects the newly identified customer-visible implementation narration.
- `web/src/components/MatterFormRemediationPanel.tsx` receives the complete linked-form copy rewrite.
- `web/src/components/MatterFormRemediationPanel.test.tsx` proves empty, setup, poor-score, review and applied copy.
- `web/src/App.tsx`, `web/src/components/FormBuilder.tsx`, `web/src/components/GovernanceAdminWorkspace.tsx`, `web/src/components/MatterDecisionResponsePanel.tsx`, `web/src/components/MatterCurrentHandoff.tsx`, `web/src/components/MatterDetailsPanel.tsx`, `web/src/components/MatterRecordWorkspace.tsx`, `web/src/components/ProgramRecordWorkspace.tsx`, `web/src/components/VendorActivationPanel.tsx`, `web/src/components/VendorDueDiligence.tsx`, `web/src/components/FormsWorkspace.tsx`, `web/src/components/matterResponsePresentation.ts`, and the listed `web/src/components/forms/**` files remove equivalent customer-visible implementation narration found by the regression.
- `web/src/document-import.css` becomes a semantic-token-only operational Imports stylesheet while preserving the scoped document-reading tokens.
- `web/ui-contract-migrations.json` makes that stylesheet part of the enforced UI contract.
- `web/scripts/ui-contract.nodecheck.mjs` proves the raw hard-coded Imports regression is rejected.
- `web/scripts/capture-ui-evidence.mjs` captures overdue Work and selected-file Imports states in both themes and narrow replacement.
- `DESIGN.md` records the distinction between lifecycle and deadline state.
- `docs/quality/rendered-ui-evidence.md` records the added evidence states.
- `docs/superpowers/issues/2026-09-03-overdue-copy-import-theme.md` records the local resolution ledger mapped to remote issue #177.

### Task 1: Add deterministic Action deadline presentation

**Files:**
- Modify: `web/src/dueDate.ts`
- Modify: `web/src/dueDate.test.ts`

- [ ] **Step 1: Write failing deadline-presentation tests**

Extend the import and add this test block:

```ts
import { actionDeadlinePresentation, selectedDateEndOfLocalDay, storedDeadlineLocalDate } from "./dueDate";

describe("actionDeadlinePresentation", () => {
  const now = new Date(2026, 8, 3, 12, 0, 0);

  it("names a passed open deadline and preserves the original date", () => {
    const due = new Date(2026, 7, 22, 17, 0, 0).toISOString();
    expect(actionDeadlinePresentation(due, false, now)).toEqual({
      dateTime: due,
      label: "Due 22 Aug 2026 · 12 days overdue",
      overdue: true,
    });
  });

  it("keeps a future open deadline scheduled", () => {
    const due = new Date(2026, 8, 7, 17, 0, 0).toISOString();
    expect(actionDeadlinePresentation(due, false, now)).toEqual({
      dateTime: due,
      label: "Due 7 Sep 2026",
      overdue: false,
    });
  });

  it("distinguishes a passed time today from a later deadline today", () => {
    const earlier = new Date(2026, 8, 3, 9, 0, 0).toISOString();
    const later = new Date(2026, 8, 3, 17, 0, 0).toISOString();
    expect(actionDeadlinePresentation(earlier, false, now).label).toBe("Due 3 Sep 2026 · overdue today");
    expect(actionDeadlinePresentation(later, false, now).label).toBe("Due 3 Sep 2026");
  });

  it("does not relabel terminal work as overdue", () => {
    const due = new Date(2026, 7, 22, 17, 0, 0).toISOString();
    expect(actionDeadlinePresentation(due, true, now).overdue).toBe(false);
    expect(actionDeadlinePresentation(due, true, now).label).toBe("Due 22 Aug 2026");
  });

  it.each([undefined, "", "not-a-date"])("keeps an invalid or missing deadline explicit", (due) => {
    expect(actionDeadlinePresentation(due, false, now)).toEqual({ label: "No action deadline", overdue: false });
  });
});
```

- [ ] **Step 2: Run the unit test and verify the new contract fails**

Run:

```powershell
npm --prefix web test -- src/dueDate.test.ts
```

Expected: FAIL because `actionDeadlinePresentation` is not exported.

- [ ] **Step 3: Implement the minimal deterministic presenter**

Append to `web/src/dueDate.ts`:

```ts
export type ActionDeadlinePresentation = {
  dateTime?: string;
  label: string;
  overdue: boolean;
};

const dayMilliseconds = 86_400_000;

function localCalendarDay(value: Date) {
  return Date.UTC(value.getFullYear(), value.getMonth(), value.getDate()) / dayMilliseconds;
}

export function actionDeadlinePresentation(
  value: string | undefined,
  terminal: boolean,
  now = new Date(Date.now()),
): ActionDeadlinePresentation {
  if (!value) return { label: "No action deadline", overdue: false };
  const due = new Date(value);
  if (!Number.isFinite(due.valueOf())) return { label: "No action deadline", overdue: false };

  const dateTime = due.toISOString();
  const date = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric" }).format(due);
  if (terminal || due.valueOf() >= now.valueOf()) return { dateTime, label: `Due ${date}`, overdue: false };

  const days = Math.max(0, localCalendarDay(now) - localCalendarDay(due));
  const age = days === 0 ? "overdue today" : `${days} day${days === 1 ? "" : "s"} overdue`;
  return { dateTime, label: `Due ${date} · ${age}`, overdue: true };
}
```

- [ ] **Step 4: Run the focused unit test**

Run: `npm --prefix web test -- src/dueDate.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit the deadline domain presentation**

```powershell
git add web/src/dueDate.ts web/src/dueDate.test.ts
git commit -m "feat(ui): derive explicit action deadline state"
```

### Task 2: Render overdue and lifecycle states separately

**Files:**
- Modify: `web/src/components/MatterActionsPanel.tsx`
- Modify: `web/src/matter-record.css`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`
- Modify: `DESIGN.md`

- [ ] **Step 1: Make the record test require visible overdue semantics**

In the Action test, set the clock and replace the old deadline assertion:

```ts
const clock = vi.spyOn(Date, "now").mockReturnValue(new Date(2026, 8, 3, 12, 0, 0).valueOf());
try {
  render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);
  expect(await screen.findByText("Program Owner", { selector: ".matter-action-meta strong" })).toBeTruthy();
  expect(screen.getByText("In progress", { selector: ".cs-status-badge" })).toBeTruthy();
  expect(screen.getByText("Overdue", { selector: ".cs-status-badge" })).toBeTruthy();
  expect(screen.getByText("Due 26 Aug 2026 · 8 days overdue", { selector: "time" })).toBeTruthy();
} finally {
  clock.mockRestore();
}
```

Add a separate test using an `IMPLEMENTED` Action fixture:

```ts
it.each(["IMPLEMENTED", "CANCELLED"] as const)("does not label a %s Action deadline overdue", async (status) => {
  vi.mocked(loadMatter).mockResolvedValue({ ...detail, actions: [{ ...detail.actions[0]!, status }] });
  render(<MatterRecordWorkspace matterID="matter-1" onBack={vi.fn()}/>);
  expect(await screen.findByText(status === "IMPLEMENTED" ? "Work completed; outcome not confirmed" : "Cancelled", { selector: ".cs-status-badge" })).toBeTruthy();
  expect(screen.queryByText("Overdue", { selector: ".cs-status-badge" })).toBeNull();
});
```

- [ ] **Step 2: Run the record test and confirm it fails on the missing badge**

Run:

```powershell
npm --prefix web test -- src/components/MatterRecordWorkspace.test.tsx -t "shows each Action owner and deadline|does not label terminal"
```

Expected: FAIL because the card has neither `StatusBadge` nor overdue text.

- [ ] **Step 3: Render the shared badges and semantic time element**

Update imports and the Action map in `MatterActionsPanel.tsx`:

```tsx
import { actionDeadlinePresentation, selectedDateEndOfLocalDay, storedDeadlineLocalDate } from "../dueDate";
import { Button, FocusedSheet, Notice, SelectField, StatusBadge, TextArea } from "./ui";

const deadline = actionDeadlinePresentation(action.due_at, terminal);

<div className="matter-action-heading">
  <div><h3 id={`matter-action-${action.id}`}>{action.title}</h3><p>{action.description}</p></div>
  <div className="matter-action-state">
    <StatusBadge tone="neutral">{statusLabel(action.status)}</StatusBadge>
    {deadline.overdue && <StatusBadge tone="error">Overdue</StatusBadge>}
  </div>
</div>
<div className="matter-action-meta">
  <span>{actionResponsibility === "ESCALATION_OWNER" ? "Escalation owner" : "Action owner"}: <strong>{ownerName}</strong></span>
  {deadline.dateTime
    ? <time className={deadline.overdue ? "matter-action-deadline is-overdue" : "matter-action-deadline"} dateTime={deadline.dateTime}>{deadline.label}</time>
    : <span className="matter-action-deadline">{deadline.label}</span>}
</div>
```

Delete the old local `formatDate` function.

- [ ] **Step 4: Add stable card layout and non-colour overdue emphasis**

Replace `.matter-action-heading > span` and add:

```css
.matter-action-state { display: flex; flex: none; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }
.matter-action-deadline.is-overdue { color: var(--coral); font-weight: 750; }

@media (max-width: 620px) {
  .matter-action-state { justify-content: flex-start; }
}
```

- [ ] **Step 5: Synchronize the interface contract**

Add under `DESIGN.md` → **States and recovery**:

```markdown
An open item whose stored deadline has passed shows **Overdue** as a separate attention state while preserving its lifecycle state and original due date. Completed and cancelled work is not newly labelled overdue. Deadline state is always named in text and never conveyed only by colour.
```

- [ ] **Step 6: Run the focused tests and UI contract**

Run:

```powershell
npm --prefix web test -- src/dueDate.test.ts src/components/MatterRecordWorkspace.test.tsx
npm --prefix web run check:ui-contracts
```

Expected: both commands PASS.

- [ ] **Step 7: Commit the Action presentation**

```powershell
git add DESIGN.md web/src/components/MatterActionsPanel.tsx web/src/components/MatterRecordWorkspace.test.tsx web/src/matter-record.css
git commit -m "feat(ui): label overdue assigned work"
```

### Task 3: Turn implementation narration into a failing regression

**Files:**
- Modify: `web/src/copyQuality.test.ts`

- [ ] **Step 1: Add narrowly scoped prohibited patterns**

Append these expressions to `productCommentary`:

```ts
  /mapped (?:missing )?(?:information|item)/i,
  /closure remain(?:s)? separate/i,
  /binding'?s score threshold/i,
  /exact (?:final )?response/i,
  /exact (?:approved )?(?:form )?revision/i,
  /immutable (?:draft|profile|template|revision)/i,
  /revision is immutable/i,
  /bounded (?:condition|simulation|response population|subject population)/i,
  /server (?:checks|score preview)/i,
  /current (?:authority|responsibility) route/i,
  /governed (?:rollback )?(?:draft|revision)/i,
  /source-backed (?:suggestion|obligation|field proposal)/i,
  /exact version/i,
  /exact change reviewers/i,
```

- [ ] **Step 2: Run the copy regression and capture the complete violation list**

Run: `npm --prefix web test -- src/copyQuality.test.ts`  
Expected: FAIL with source locations in the affected Matter, Program and Forms components. Preserve this output in the issue-resolution ledger created in Task 6.

### Task 4: Rewrite the complete affected workflow copy

**Files:**
- Modify: `web/src/components/MatterFormRemediationPanel.tsx`
- Modify: `web/src/components/MatterFormRemediationPanel.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/FormBuilder.tsx`
- Modify: `web/src/components/GovernanceAdminWorkspace.tsx`
- Modify: `web/src/components/MatterDecisionResponsePanel.tsx`
- Modify: `web/src/components/MatterCurrentHandoff.tsx`
- Modify: `web/src/components/MatterDetailsPanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.tsx`
- Modify: `web/src/components/VendorActivationPanel.tsx`
- Modify: `web/src/components/VendorDueDiligence.tsx`
- Modify: `web/src/components/FormsWorkspace.tsx`
- Modify: `web/src/components/matterResponsePresentation.ts`
- Modify: `web/src/components/forms/CommunicationsView.tsx`
- Modify: `web/src/components/forms/DistributionComposer.tsx`
- Modify: `web/src/components/forms/FormAIComposer.tsx`
- Modify: `web/src/components/forms/FormPoliciesView.tsx`
- Modify: `web/src/components/forms/FormPolicyEditor.tsx`
- Modify: `web/src/components/forms/FormProposalReview.tsx`
- Modify: `web/src/components/forms/builder/AdvancedScoringEditor.tsx`
- Modify: `web/src/components/forms/builder/FormApprovalSheet.tsx`
- Modify: `web/src/components/forms/filters/AdvancedFilterEditor.tsx`
- Modify affected component tests containing old visible wording.

- [ ] **Step 1: Make linked-form tests require operational wording**

Add assertions for the empty, setup, poor-score, review and applied states:

```ts
expect(await screen.findByText("Send an approved form for the outstanding items. Review the response before updating this issue.")).toBeTruthy();
expect(screen.getByText("No approved form has been sent for the outstanding items in this issue.", { exact: false })).toBeTruthy();

expect(await screen.findByText("Choose which form answer supplies each outstanding item. These choices are fixed after the request is sent.")).toBeTruthy();

expect(await screen.findByText("The response score is below the approved threshold. Review it or request a correction; this issue remains open.")).toBeTruthy();

expect(screen.getByText("Explain why this response answers the outstanding items.")).toBeTruthy();

expect(await screen.findByText("Information added · confirm the result before closing")).toBeTruthy();
```

For the applied-state assertion, use this complete fixture in its own test:

```ts
it("states the remaining outcome work after information is added", async () => {
  vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
    binding,
    request: { id: "request-1", title: "Restore source evidence", status: "COMPLETED", deadline: "2026-09-12T23:59:59Z" },
    response: { id: "response-revision-1", revision: 3, current: true, state: "FINAL", completed_at: "2026-09-03T10:00:00Z" },
    application: { id: "application-1", response_revision_id: "response-revision-1", matter_version: 8, applied_at: "2026-09-03T10:05:00Z" },
    next_action: "Check outcome",
  }]);
  render(<MatterFormRemediationPanel aggregate={aggregate} operations={[reviewerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);
  expect(await screen.findByText("Information added · confirm the result before closing")).toBeTruthy();
});
```

Add the remaining state tests, importing `ApiError` from `../http`:

```ts
it("keeps the named linked-request population visible while loading", () => {
  vi.mocked(loadMatterFormRemediations).mockReturnValue(new Promise(() => undefined));
  render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);
  expect(screen.getByText("Checking linked requests and responses…")).toBeTruthy();
});

it("names a linked-request load failure and its affected action", async () => {
  vi.mocked(loadMatterFormRemediations).mockRejectedValue(new Error("Request service unavailable"));
  render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Linked form action could not be completed.");
  expect(screen.getByRole("alert").textContent).toContain("Request service unavailable");
});

it("opens a sent linked request without asking for another evidence record", async () => {
  vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
    binding,
    request: { id: "request-1", title: "Restore source evidence", status: "READY", deadline: "2026-09-12T23:59:59Z" },
    next_action: "Open response",
  }]);
  const openRequest = vi.fn();
  render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onOpenRequest={openRequest} onMappingsChange={vi.fn()}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open response" }));
  expect(openRequest).toHaveBeenCalledWith("request-1");
});

it("preserves the review basis when the issue changed", async () => {
  vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
    binding,
    request: { id: "request-1", title: "Restore source evidence", status: "COMPLETED", deadline: "2026-09-12T23:59:59Z" },
    response: { id: "response-revision-1", revision: 3, current: true, state: "FINAL", completed_at: "2026-09-03T10:00:00Z" },
    next_action: "Review evidence",
  }]);
  vi.mocked(applyMatterFormRemediation).mockRejectedValue(new ApiError(409, "changed", "version_conflict"));
  render(<MatterFormRemediationPanel aggregate={aggregate} operations={[reviewerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);
  const basis = await screen.findByLabelText("Review basis");
  fireEvent.change(basis, { target: { value: "The response answers both outstanding items with current evidence." } });
  fireEvent.click(screen.getByRole("button", { name: "Apply response" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Reload the issue before applying the response.");
  expect((basis as HTMLTextAreaElement).value).toContain("answers both outstanding items");
});
```

- [ ] **Step 2: Run linked-form tests and verify the new wording fails**

Run: `npm --prefix web test -- src/components/MatterFormRemediationPanel.test.tsx`  
Expected: FAIL on the new production-copy assertions.

- [ ] **Step 3: Rewrite every linked-form state**

Use these exact customer-visible strings in `MatterFormRemediationPanel.tsx`:

```tsx
<p>Send an approved form for the outstanding items. Review the response before updating this issue.</p>

<p>No approved form has been sent for the outstanding items in this issue. {ownerCanAct ? "Send a form to collect the information." : "The current issue owner can send a form."}</p>

<span>{state.next_action} · {state.binding.mappings.length} outstanding item{state.binding.mappings.length === 1 ? "" : "s"} covered</span>

<Notice tone="warning">The response score is below the approved threshold. Review it or request a correction; this issue remains open.</Notice>

<TextArea label="Review basis" value={rationaleByBinding[state.binding.id] ?? ""} onChange={(value) => setRationaleByBinding((current) => ({ ...current, [state.binding.id]: value }))} description="Explain why this response answers the outstanding items."/>

<span className="status-pill success">Information added · confirm the result before closing</span>

<p>Choose which form answer supplies each outstanding item. These choices are fixed after the request is sent.</p>
```

Keep the existing API identifiers, immutable response selection and authority checks unchanged.

- [ ] **Step 4: Rewrite the remaining scanner violations with business-language equivalents**

Use the following exact replacements:

```text
Review the response package and complete the step assigned to you.
No person or role is currently assigned to this issue.
Choose an eligible person and record why responsibility is changing.
Checking who can act on this issue.
Issue details remain visible while assignments are loading.
Issue assignments could not be checked. Details remain visible, but changes are disabled until assignments can be confirmed.
Some assignee names could not be loaded. Recorded assignments remain visible, and available actions continue to use the current assignments.
Checking who can act on this Program.
Program details remain visible while assignments are loading.
Program assignments could not be checked. Details and recorded owners remain visible, but changes are disabled until assignments can be confirmed.
Changes remain unavailable until the current approval route is confirmed.
You are not permitted to record this activation decision.
Starter copied to a new draft. Review the questions and quality checks before requesting approval.
Form draft saved as a new version.
The current form draft could not be loaded.
Create a new form draft or open Forms to review and approve an existing template before starting this vendor review.
New profile version
Saving creates a new profile version with its own effective period. Earlier versions remain in history.
Create the first message-template draft before sending form messages.
Select an existing message template or create the first draft.
The selected form version cannot change after sending.
Describe the form to create or the specific changes reviewers need.
Create an issue from a completed response that meets an approved concern threshold. Simulation and independent approval are required before activation.
Create a draft to select the approved form, eligible subjects, issue handling and outcome check.
Simulate this policy version before requesting approval.
Policy approved. A permitted user must still activate it.
A rollback draft was created for review.
Define which completed responses create an issue. Activation requires a current simulation and independent approval.
Choose one approved form version and the subject types this policy covers.
Preparing proposed fields from the imported document…
The imported file remains unchanged while candidate fields are prepared.
Choose the field changes to include before creating a draft.
Set the points awarded when a response matches a condition.
Save this form before previewing how responses will be scored.
This does not activate the form. A different approver must review and approve the version shown here.
New draft version
Your changes will be saved as a new draft version before it is submitted.
Combine conditions
Use fields from this form. You can add up to 12 conditions across 3 levels.
This records the selected response state. Assigned work and outcome checks are unchanged.
This records the selected issue state. Assigned work and outcome checks are unchanged.
Import documents, compare their obligations with current Programs, controls and evidence, then review proposed updates.
Issue created and linked to this imported obligation.
```

Do not replace API field names, domain event names, test descriptions or specialist history values unless they are rendered to customers.

- [ ] **Step 5: Run the copy and affected component suites**

Run:

```powershell
npm --prefix web test -- src/copyQuality.test.ts src/components/MatterFormRemediationPanel.test.tsx src/components/MatterRecordWorkspace.test.tsx src/components/FormsWorkspace.test.tsx src/components/forms
```

Expected: PASS with no old implementation-narration match.

- [ ] **Step 6: Commit the production-copy pass**

```powershell
git add web/src/copyQuality.test.ts web/src/App.tsx web/src/components
git commit -m "fix(copy): use production language in governed workflows"
```

### Task 5: Migrate the complete Imports workspace to semantic theme tokens

**Files:**
- Modify: `web/src/document-import.css`
- Modify: `web/ui-contract-migrations.json`
- Modify: `web/scripts/ui-contract.nodecheck.mjs`
- Modify: `web/src/components/DocumentImportWorkspace.test.tsx`

- [ ] **Step 1: Add Imports to the enforced CSS contract**

Add `"src/document-import.css"` to `migratedCss` and add this node test:

```js
test("document import surfaces reject local dark palette values", async () => {
  const source = await read("src/document-import.css");
  const diagnostics = validateCssSource({ file: "src/document-import.css", source });
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
  assert.doesNotMatch(source, /rgba?\(/i);
  assert.match(source, /\.document-import-form\s*\{[^}]*background:\s*var\(--surface-2\)/is);
});
```

- [ ] **Step 2: Run the contract and verify the existing dark values fail**

Run: `npm --prefix web run check:ui-contracts`  
Expected: FAIL on the hard-coded dark backgrounds, borders, shadows and numeric radii in `document-import.css`.

- [ ] **Step 3: Replace the Imports operational palette**

Apply these complete semantic mappings throughout `document-import.css`:

```css
.document-import-form { background: var(--surface-2); }
.document-import-row.active { border-color: color-mix(in srgb, var(--cyan) 40%, var(--border)); background: var(--cyan-soft); }
.document-metadata div { background: var(--surface-detail); }
.document-limitations { border-bottom-color: color-mix(in srgb, var(--amber) 28%, var(--border)); }
.proposal-title mark { background: var(--violet-soft); }
.proposal-card blockquote { background: color-mix(in srgb, var(--cyan) 5%, var(--surface)); }
.coverage-assessment { border-color: color-mix(in srgb, var(--cyan) 24%, var(--border)); background: var(--surface-2); }
.coverage-scoreboard { background: var(--surface); }
.coverage-scoreboard .coverage-primary { background: var(--cyan-soft); }
.coverage-callout.warning { border-color: color-mix(in srgb, var(--amber) 36%, var(--border)); background: var(--amber-soft); }
.coverage-callout.danger { border-color: color-mix(in srgb, var(--coral) 38%, var(--border)); background: var(--coral-soft); }
.coverage-notice { background: var(--cyan-soft); }
.coverage-filters { background: var(--surface); }
.coverage-filters button[aria-pressed="true"] { background: var(--cyan-soft); }
.coverage-candidate { background: var(--surface); }
.coverage-candidate.classification-gap,
.coverage-candidate.classification-mapped-control-gap,
.coverage-candidate.classification-mapped-no-current-evidence { border-left-color: var(--amber); }
.coverage-candidate.classification-verified-coverage { border-left-color: var(--cyan); }
.coverage-candidate > blockquote,
.coverage-match { border-color: color-mix(in srgb, var(--cyan) 24%, var(--border)); background: color-mix(in srgb, var(--cyan) 5%, var(--surface)); }
.coverage-chain span.active { border-color: color-mix(in srgb, var(--cyan) 38%, var(--border)); background: var(--cyan-soft); }
.coverage-no-match { background: var(--amber-soft); }
.document-degradations { border-bottom-color: color-mix(in srgb, var(--amber) 32%, var(--border)); }
.document-degradations > div { border-color: color-mix(in srgb, var(--amber) 30%, var(--border)); background: var(--amber-soft); }
```

Replace every numeric `border-radius` with the closest existing token:

```css
border-radius: var(--cs-primitive-radius-6);
border-radius: var(--cs-primitive-radius-10);
border-radius: var(--cs-primitive-radius-14);
border-radius: var(--cs-primitive-radius-full);
```

Replace the inspector shadow with `box-shadow: var(--cs-shadow-raised);`. Preserve the `--document-*` token scope on `.document-import-inspector`; it is the intentional document-reading surface.

- [ ] **Step 4: Add a component assertion for the semantic intake class and preserved recovery**

In `DocumentImportWorkspace.test.tsx`, keep the existing upload-recovery test and add:

```ts
it("uses the semantic intake surface while preserving entered import work", async () => {
  vi.mocked(importDocument).mockRejectedValue(new Error("Upload interrupted"));
  const { container } = render(<DocumentImportWorkspace/>);
  await screen.findByRole("heading", { name: "regulatory-notice.md" });
  await openImport();
  expect(container.querySelector(".document-import-form")).toBeTruthy();
  const file = new File(["notice"], "retry-notice.pdf", { type: "application/pdf" });
  fireEvent.change(screen.getByLabelText("Document"), { target: { files: [file] } });
  fireEvent.change(screen.getByRole("textbox", { name: "What should reviewers look for?" }), { target: { value: "Review payment requirements" } });
  fireEvent.click(screen.getByRole("button", { name: "Import document" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Upload interrupted");
  expect(screen.getByText(/retry-notice\.pdf/)).toBeTruthy();
});
```

- [ ] **Step 5: Run contract, component, axe and build checks**

Run:

```powershell
npm --prefix web run check:ui-contracts
npm --prefix web test -- src/components/DocumentImportWorkspace.test.tsx
npm --prefix web run typecheck
npm --prefix web run build
```

Expected: all commands PASS.

- [ ] **Step 6: Commit the semantic Imports migration**

```powershell
git add web/src/document-import.css web/ui-contract-migrations.json web/scripts/ui-contract.nodecheck.mjs web/src/components/DocumentImportWorkspace.test.tsx
git commit -m "fix(ui): restore imports theme parity"
```

### Task 6: Add rendered states and complete the resolution ledger

**Files:**
- Modify: `web/scripts/capture-ui-evidence.mjs`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Create: `docs/superpowers/issues/2026-09-03-overdue-copy-import-theme.md`

- [ ] **Step 1: Add deterministic overdue and Imports captures**

Add these capture definitions:

```js
{ name: "176-matter-overdue-action-light-1440x900", route: "#work/matters/matter-gaid-change", title: "Work", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "matter-overdue-action", expectText: "Overdue" },
{ name: "177-matter-overdue-action-dark-mobile-390x844", route: "#work/matters/matter-gaid-change", title: "Work", theme: "dark", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, state: "matter-overdue-action-mobile", expectText: "Overdue" },
```

Replace `captureImportSelection()` with a parameterized implementation and call it for:

```js
await captureImportSelection("178-import-selected-light-1440x900", "light", { width: 1440, height: 900 });
await captureImportSelection("179-import-selected-dark-1440x900", "dark", { width: 1440, height: 900 });
await captureImportSelection("180-import-selected-light-mobile-390x844", "light", { width: 390, height: 844 }, true);
```

The function must continue to select `outsourcing-policy.pdf`, assert the replace action remains inside the dropzone, call `assertNoHorizontalOverflow`, and save/record each state.

- [ ] **Step 2: Build and run the rendered evidence suite**

Run in separate terminals:

```powershell
npm --prefix web run build:evidence
npm --prefix web run preview:evidence -- --host 127.0.0.1
```

Then:

```powershell
$env:PAGE_URL='http://127.0.0.1:4173'
$env:UI_EVIDENCE_DIR='ui-evidence/issue-177'
node web/scripts/capture-ui-evidence.mjs
```

Expected: the manifest reports no failure and contains captures 176–180 with no horizontal overflow.

- [ ] **Step 3: Inspect the five PNGs and repair the highest-impact defect**

Inspect:

```text
ui-evidence/issue-177/176-matter-overdue-action-light-1440x900.png
ui-evidence/issue-177/177-matter-overdue-action-dark-mobile-390x844.png
ui-evidence/issue-177/178-import-selected-light-1440x900.png
ui-evidence/issue-177/179-import-selected-dark-1440x900.png
ui-evidence/issue-177/180-import-selected-light-mobile-390x844.png
```

Review object/action clarity, both Action states, due-date legibility, form/dropzone contrast, field boundaries, primary action contrast, mobile replacement, clipping and focus. Fix the highest-impact problem in its owning component or stylesheet, rerun its focused tests, and recapture the affected PNG.

- [ ] **Step 4: Update rendered-evidence documentation**

Add to `docs/quality/rendered-ui-evidence.md`:

```markdown
- open Action cards show lifecycle and overdue deadline as separate text states in desktop and mobile Work renders;
- selected-file Imports intake is captured in light and dark desktop themes and light mobile replacement, including readable fields, dropzone replacement and the explicit import action.
```

- [ ] **Step 5: Create the local resolution ledger**

Write `docs/superpowers/issues/2026-09-03-overdue-copy-import-theme.md` with:

```markdown
# Remote issue #177 resolution ledger

## Scope

- explicit overdue Action presentation;
- production-copy audit for affected Work, Program and Forms workflows;
- semantic Imports theme parity.

## Before-state causes

- `MatterActionsPanel` formatted the date without deriving deadline state;
- affected customer strings described mappings, immutable revisions, bounded logic and server behavior;
- `document-import.css` owned dark RGBA surfaces outside the theme system.

## Verification

Record the final commit, focused test commands, full web suite, UI contract, typecheck, production build, rendered-evidence manifest path and the highest-impact visual repair.

## Remaining work

List only failures or copy findings outside the approved affected-workflow scope, each with a remote issue link. Use **None found** only after the complete source scan and rendered review pass.
```

- [ ] **Step 6: Commit rendered proof and the ledger**

```powershell
git add web/scripts/capture-ui-evidence.mjs docs/quality/rendered-ui-evidence.md docs/superpowers/issues/2026-09-03-overdue-copy-import-theme.md
git commit -m "test(ui): prove overdue and imports theme states"
```

### Task 7: Run release gates and update remote issue #177

**Files:**
- Modify only files required by a failing release gate.

- [ ] **Step 1: Run the complete web release suite**

```powershell
npm --prefix web test
npm --prefix web run check:runtime-truth
npm --prefix web run check:ui-contracts
npm --prefix web run typecheck
npm --prefix web run build
```

Expected: every command exits 0.

- [ ] **Step 2: Run repository whitespace and branch checks**

```powershell
git diff --check origin/main...HEAD
git status --short
git log --oneline origin/main..HEAD
```

Expected: no whitespace errors, no uncommitted files, and only the issue-177 commits.

- [ ] **Step 3: Update the ledger with exact evidence**

Replace the Verification instructions in the ledger with the actual commit SHAs, command results, manifest path and inspected PNG names. If a release gate fails, record the failure and fix it before continuing.

- [ ] **Step 4: Commit the final evidence record**

```powershell
git add docs/superpowers/issues/2026-09-03-overdue-copy-import-theme.md
git commit -m "docs: record issue 177 verification"
```

- [ ] **Step 5: Push and update the remote tracker**

```powershell
git push origin codex/overdue-copy-import-fix
gh issue comment 177 --body "Implementation and verification are complete on branch codex/overdue-copy-import-fix. The resolution ledger is docs/superpowers/issues/2026-09-03-overdue-copy-import-theme.md. A pull request will link the exact test and rendered evidence before this issue is closed."
```

Expected: push succeeds and the comment appears on issue #177. Do not close the issue until the pull request is merged and the deployed revision is verified.
