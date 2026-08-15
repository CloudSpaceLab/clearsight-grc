# Product Copy Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the root contributor rules prevent customer-facing copy from regressing into product commentary, implementation narration or blocking guidance.

**Architecture:** Add one normative subsection to the existing `Human-friendly enterprise copy` section in the root `AGENTS.md`. The subsection defines the semantic admission test, complete source scope, prohibited narration, onboarding and acronym requirements, and mandatory verification using the existing copy regression and rendered workflow checks.

**Tech Stack:** Markdown contributor rules, Vitest copy-quality regression, Git diff validation.

---

### Task 1: Add the mandatory customer-facing copy gate

**Files:**
- Modify: `AGENTS.md:46-65`
- Reference: `docs/superpowers/specs/2026-08-15-product-copy-governance-design.md`
- Reference: `web/src/copyQuality.test.ts`

- [x] **Step 1: Confirm the edit target is the root rule file**

Run:

```powershell
Get-ChildItem -Path . -Filter AGENTS.md -Recurse -Force | Select-Object -ExpandProperty FullName
```

Expected: only `C:\dev\clearsight-grc\AGENTS.md`, so the new rule governs the complete repository.

- [x] **Step 2: Add the normative copy gate after the existing copy bullets**

Insert this Markdown immediately before `## UI design proof`:

```markdown
### Customer-facing copy gate

- This gate applies to every customer-visible string from React components, server and API responses, onboarding, demo fixtures, empty states, errors, notifications, tooltips, labels, accessibility text and illustration descriptions.
- Every customer-facing sentence MUST identify a business object or task; state a condition, source, owner, deadline or freshness; explain why the current bank role must act; state the next action and result; or explain a limitation, consequence or recovery step. If it does none of these, remove or rewrite it.
- Copy MUST address the bank user at the point of work. It MUST NOT compare ClearSight with another product category, defend a product or design decision, narrate internal architecture, or teach implementation terminology that is unnecessary for the task.
- Product-review commentary is prohibited, including references to a “generic dashboard,” an “exact record,” canonical or bounded views, projections, authoritative servers, internal resolution behavior, implementation guarantees, and equivalent rewordings. This is a semantic rule; passing a fixed phrase scan is not sufficient.
- Headings name the task, record, state or decision. Buttons use a direct verb and name the immediate result. Supporting text adds status, context, consequence or recovery information instead of repeating the heading.
- Familiar role and governance acronyms retain their established casing, including CRO, CCO, CISO and GRC.
- Guides orient users to work in concise business language. They MUST remain optional, dismissible, accessible and non-blocking, including when progress cannot be saved.
- Simpler wording MUST NOT weaken authority, evidence, legal-scope, uncertainty or compliance limitations.
- A copy change MUST review the complete affected workflow and every relevant source, not only the edited phrase. Phrase-by-phrase substitution is insufficient when equivalent commentary remains elsewhere.
- When a new class of product narration can be detected reliably, extend `web/src/copyQuality.test.ts` with a semantic pattern that avoids broad false positives. The pattern list is a regression aid, not the complete writing standard.
- Before completion, run the copy-quality regression and affected workflow tests, render every materially affected workspace at relevant viewport sizes, and confirm that guides, notices and errors do not block primary actions.
```

- [x] **Step 3: Check the rule for ambiguity and formatting problems**

Run:

```powershell
$copyGate = Get-Content -Raw AGENTS.md
if ($copyGate -notmatch '### Customer-facing copy gate') { throw 'Copy gate missing' }
if ($copyGate -notmatch 'semantic rule; passing a fixed phrase scan is not sufficient') { throw 'Semantic requirement missing' }
if ($copyGate -notmatch 'server and API responses') { throw 'Source scope incomplete' }
if ($copyGate -notmatch 'optional, dismissible, accessible and non-blocking') { throw 'Guide requirement missing' }
git diff --check
```

Expected: no output from `git diff --check` and no PowerShell exception.

- [x] **Step 4: Run the existing automated copy gate**

Run:

```powershell
npm test -- --run src/copyQuality.test.ts
```

Working directory: `C:\dev\clearsight-grc\web`

Expected: `src/copyQuality.test.ts` passes with one test and no detected customer-facing product commentary.

- [x] **Step 5: Review the final diff and commit the rule**

Run:

```powershell
git diff -- AGENTS.md
git status --short
git add AGENTS.md docs/superpowers/plans/2026-08-15-product-copy-governance.md
git commit -m "docs: enforce customer-facing copy gate"
```

Expected: the diff contains only the approved root rule addition and plan tracking; the commit completes successfully.
