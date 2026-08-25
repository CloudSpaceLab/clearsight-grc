# ClearSight Business Demo Deck Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create and verify a 15-slide premium PowerPoint business and product demonstration deck for ClearSight GRC, aimed at Nigerian bank decision-makers and based on the supplied FraudSniper visual reference.

**Architecture:** Use the FraudSniper PPTX as the source template, map every ClearSight slide to one inherited source slide, create a starter deck by duplicating those slides, and edit inherited elements with `@oai/artifact-tool`. Capture current ClearSight static-demo screens for product proof, keep all build artifacts in a temporary workspace, and deliver one editable final PPTX after template-fidelity, content, overflow, and full-slide visual QA.

**Tech Stack:** PowerPoint/PPTX, `@oai/artifact-tool`, bundled Node.js and Python presentation utilities, LibreOffice rendering, Vite static demo, browser screenshots, and official web sources for any current regulatory statements.

---

### Task 1: Establish the build workspace and inspect the reference deck

**Files:**
- Create: `.codex-tmp/clearsight-business-deck-20260825/unzip-shim.mjs`
- Create: `.codex-tmp/clearsight-business-deck-20260825/unzip.cmd`
- Create: `.codex-tmp/clearsight-business-deck-20260825/template-audit.txt`
- Create: `.codex-tmp/clearsight-business-deck-20260825/source-notes.txt`
- Source: `.codex-tmp/clearsight-business-deck-20260825/fraudsniper-reference.pptx`

- [ ] **Step 1: Record the runtime paths supplied by `codex_app__load_workspace_dependencies`**

Use these command-scoped values for every presentation command:

```text
RUNTIME_NODE=C:\Users\Son\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe
RUNTIME_NODE_MODULES=C:\Users\Son\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\node_modules
RUNTIME_BIN_DIR=C:\Users\Son\.cache\codex-runtimes\codex-primary-runtime\dependencies\bin\override
```

- [ ] **Step 2: Add a local compatibility shim for the template inspector's `unzip -Z1` and `unzip -p` calls**

Implement the shim with bundled Node.js and `node:zlib`/ZIP package support available in `RUNTIME_NODE_MODULES`. The shim must only list or stream ZIP entries; it must not mutate the source PPTX. Put `unzip.cmd` first on `PATH` for inspection commands.

- [ ] **Step 3: Run the required source-deck inventory**

Run:

```powershell
& $env:RUNTIME_NODE "$env:SKILL_DIR\template_following_scripts\inspect_template_deck.mjs" --workspace $env:TMP_DIR --pptx "$env:TMP_DIR\fraudsniper-reference.pptx"
```

Expected: 23 source-slide renders, layout JSON, extracted media, font evidence, `template-inspect.ndjson`, and `template-manifest.json` under the temporary workspace.

- [ ] **Step 4: Write the template audit**

Record the source deck's reusable patterns: minimal cover, thesis/market-context slide, executive-summary split, architecture/process slide, feature explanation, comparison/table, roadmap/timeline, product-screenshot slide, and close. Record inherited footer/page-number behavior, typography, spacing, source logo objects to remove, and the approved ClearSight color deviations.

- [ ] **Step 5: Verify the inspection inventory**

Run a file-count check and open the full contact sheet. Expected: all 23 slides represented, no missing renders, and enough editable inherited frames to support all 15 output slides.

### Task 2: Capture current ClearSight product visuals

**Files:**
- Create: `.codex-tmp/clearsight-business-deck-20260825/screens/today.png`
- Create: `.codex-tmp/clearsight-business-deck-20260825/screens/programs.png`
- Create: `.codex-tmp/clearsight-business-deck-20260825/screens/work.png`
- Create: `.codex-tmp/clearsight-business-deck-20260825/screens/imports.png`
- Create: `.codex-tmp/clearsight-business-deck-20260825/screenshot-notes.txt`

- [ ] **Step 1: Start the current React static stakeholder demo**

Run the Vite application with `VITE_STATIC_DEMO=true`, `VITE_ENABLE_SAMPLE_DATA=true`, and the existing static-demo transport. Use the bundled Node runtime and package directory; do not install packages.

- [ ] **Step 2: Capture the four product states at a consistent desktop viewport**

Capture Today, Programs, Work, and Imports at 1440×900. Each screenshot must show the visible `Reference data` or stakeholder-demo context where the product supplies it and must avoid open developer tools, browser chrome, transient toasts, or clipped side panels.

- [ ] **Step 3: Inspect every screenshot at original size**

Reject any capture with loading placeholders, stale modal state, hidden primary actions, low-resolution scaling, or obsolete content. Record the route, viewport, visible sample-data label, and capture time in `screenshot-notes.txt`.

### Task 3: Finalize source-grounded copy and provenance

**Files:**
- Create: `.codex-tmp/clearsight-business-deck-20260825/content.txt`
- Modify: `.codex-tmp/clearsight-business-deck-20260825/source-notes.txt`

- [ ] **Step 1: Write final audience-facing copy for all 15 slides**

Use the approved slide sequence from `docs/superpowers/specs/2026-08-25-clearsight-business-demo-deck-design.md`. Keep one claim per slide, no unexplained percentages, no speculative ROI, no autonomous compliance claims, and no invented logo treatment.

- [ ] **Step 2: Verify Nigerian references against official current sources**

Use official NDPC and Nigerian legal/regulatory sources for any visible current claim. Keep detailed legal statements out of visible copy unless necessary. Label the four repository journeys as reference data and state that they are not legal advice.

- [ ] **Step 3: Create the source ledger and notes blocks**

For each slide, record repository source files and any official external URL. Draft a `[Sources]` block for speaker notes on every slide containing an external claim or asset. Product-screen slides must identify the current static-demo route and sample-data status.

- [ ] **Step 4: Run the copy gate**

Search final slide copy for internal codes, unexplained `Matter` terminology, inflated claims, generic AI language, competitive chest-beating, missing reference-data labels, and unresolved placeholder words. Expected: no violations.

### Task 4: Map the 15-slide narrative to inherited source slides

**Files:**
- Create: `.codex-tmp/clearsight-business-deck-20260825/template-frame-map.json`
- Create: `.codex-tmp/clearsight-business-deck-20260825/deviation-log.txt`
- Create: `.codex-tmp/clearsight-business-deck-20260825/edit-plan.txt`

- [ ] **Step 1: Select the source-slide pattern for every output slide**

Use source slide 1 or 2 for the cover; source slides 3–5 for context and executive thesis; source slides 6–9 for Programs, issues/changes, evidence, and authority; source slides 12–13 for Nigerian-bank reference scope; source slides 14, 17, or 18 for the pilot; source slides 19–21 for operating-chain or architecture diagrams; and source slides 22–23 for product proof and close. Adjust selections only when the inspection proves a different inherited frame fits better.

- [ ] **Step 2: Define exact edit targets**

For each output slide, list every inherited shape ID to rewrite, replace, delete, or fill. Explicitly delete FraudSniper logos, FraudSniper wording, July 2025 labels, unused source footers, and any empty structural placeholders. Preserve Cloudspace Technologies Ltd attribution using the source deck's authentic Cloudspace asset where available.

- [ ] **Step 3: Validate the frame map**

Run the template plan validator through `prepare_template_starter_deck.mjs`. Expected: all 15 output slides have a source slide, every edit target resolves to an inherited element, and no unbounded overlay additions are requested.

- [ ] **Step 4: Generate and inspect the starter deck**

Create `template-starter.pptx`, starter renders, layout JSON, and contact sheet. Confirm the 15-slide order and inherited layouts before any content edits.

### Task 5: Author the ClearSight deck with artifact-tool

**Files:**
- Create: `.codex-tmp/clearsight-business-deck-20260825/build-clearsight-deck.mjs`
- Create: `ClearSight GRC - Business and Product Demo.pptx`

- [ ] **Step 1: Read the required artifact-tool authoring references**

Read `API_QUICK_START.md`, `API_DOCS.md`, `master.spec.md`, `layout.spec.md`, `inspect.md`, and `cookbook/imported-deck.md` from the Presentations skill directory.

- [ ] **Step 2: Link the bundled Node package directory into the temporary build workspace**

Create a Windows junction named `node_modules` pointing to `RUNTIME_NODE_MODULES`. Do not modify the bundled dependency directory.

- [ ] **Step 3: Mark the artifact operation exactly once**

Run:

```powershell
& $env:RUNTIME_NODE "$env:SKILL_DIR\container_tools\mark_artifact_operation_started.mjs" --operation-kind create --expected-output-count 1 --output-format pptx
```

Expected: success before the first authoring command.

- [ ] **Step 4: Implement inherited-element edits**

Import `template-starter.pptx`, edit the mapped text, images, tables, diagrams, footer content, and speaker notes in place, preserve master/layout relationships, and export the final deck. Keep template font family, sizes, line spacing, insets, and vertical alignment; shorten copy or remap a slide rather than shrinking text.

- [ ] **Step 5: Apply the approved ClearSight color deviations**

Change coral brand accents to ClearSight cyan or violet where they represent active information or governance. Retain amber, green, and coral only for their documented semantic states. Record every intentional color or asset departure in `deviation-log.txt`.

- [ ] **Step 6: Insert authentic product screenshots into inherited media frames**

Use the captured Today, Programs, Work, and Imports images without distortion. Crop to preserve the actor, record state, reason, and next action. Do not reuse one screenshot on more than one slide.

- [ ] **Step 7: Export the final PPTX**

Save exactly one final artifact to `C:\dev\clearsight-grc\ClearSight GRC - Business and Product Demo.pptx`.

### Task 6: Run content, fidelity, and visual QA

**Files:**
- Create: `.codex-tmp/clearsight-business-deck-20260825/final-render/slide-1.png` through `slide-15.png`
- Create: `.codex-tmp/clearsight-business-deck-20260825/qa-ledger.txt`
- Modify: `.codex-tmp/clearsight-business-deck-20260825/build-clearsight-deck.mjs`
- Modify: `ClearSight GRC - Business and Product Demo.pptx`

- [ ] **Step 1: Render every final slide and create a montage**

Use the bundled presentation renderer. Expected: 15 full-size PNGs and one contact sheet.

- [ ] **Step 2: Run structural tests**

Run `slides_test.py`, the template fidelity checker, an OOXML empty-placeholder scan, and text extraction or inspection available through artifact-tool. Expected: no canvas overflow, unresolved structural placeholders, unintended template deletions, or leftover FraudSniper content.

- [ ] **Step 3: Inspect every slide individually**

Check hierarchy, title wrapping, clipping, alignment, minimum margins, text-image spacing, contrast, screenshot crop, diagram connector routing, footer consistency, source notes, reference-data labels, and slide-to-slide pacing. Record at least one improvement opportunity in `qa-ledger.txt` before the first repair pass.

- [ ] **Step 4: Fix the highest-impact issues**

Edit inherited elements or remap the affected slide. Do not cover a template problem with an overlay and do not shrink type below the inherited size.

- [ ] **Step 5: Re-export and re-render affected slides**

Repeat structural and visual checks. Expected: a full second pass with no new unintended overlap, clipping, placeholder, fidelity, or copy issues.

- [ ] **Step 6: Run final product-copy verification**

Run `npm test -- --run web/src/copyQuality.test.ts` or the repository's exact affected copy-quality command if the test lives under the web package. Confirm the presentation uses the same semantic standard even though the PPTX is not compiled by that test.

### Task 7: Deliver the verified presentation

**Files:**
- Verify: `ClearSight GRC - Business and Product Demo.pptx`

- [ ] **Step 1: Confirm final file integrity**

Verify the PPTX opens, contains 15 slides, includes speaker notes where sources are required, and has a non-zero recent modification timestamp.

- [ ] **Step 2: Confirm no extra deliverables were created at the output location**

Only the final PPTX should sit at the requested output location; previews and scratch files remain under `.codex-tmp`.

- [ ] **Step 3: Hand off the presentation**

Provide one output citation for the final deck and summarize the narrative, authentic product visuals, source treatment, and completed QA cycle.
