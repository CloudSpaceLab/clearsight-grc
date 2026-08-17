# Demo Account and Document Import Experience Design

**Status:** Approved direction
**Date:** 12 August 2026

## Problem

The demo login repeats usernames and the same password across every role card, then repeats those credentials again in a second form. The persistent floating role-switch button is visually detached from the signed-in identity and returning to the full sign-in screen makes comparison between roles feel heavier than necessary.

The Imports workspace gives a permanent, dense intake form the same visual weight as document review. It does not state the configured 20 MiB limit before submission, and its PDF language does not make the current boundary obvious enough: PDFs are safely stored and hashed, but this build deliberately has no approved PDF text extractor or OCR adapter.

## Considered approaches

### 1. Restyle the existing screens

Keep the two-column credential form, card catalogue and floating switch button, changing only spacing and colors. This is low risk but leaves the duplicated information and full-page switch interruption intact.

### 2. Compact role chooser plus progressive import intake

Represent each role once with a compact selectable row, show one dominant `Continue as …` action, and hide the shared demo credentials from the normal path. Put the signed-in account control in a compact fixed menu that securely logs out before another role is selected. Collapse document intake behind a clear `Import document` action when records already exist, while keeping it open for an empty workspace. Show file format and 20 MiB constraints before upload, processing status after the durable receipt, and a precise stored-only PDF outcome.

### 3. In-place account impersonation and a multi-step upload wizard

Add an API that swaps identities without logout and turn imports into a modal wizard. This could reduce clicks further, but identity swapping risks retaining actor-scoped UI state and a wizard is disproportionate to three inputs.

**Decision:** Approach 2. It removes duplicated information, preserves the security boundary that switching identities unmounts the application, and applies progressive disclosure without introducing a new workflow framework.

## Demo account experience

- The sign-in page shows a concise role list. Each option contains the role name and plain-language role coverage only; it does not repeat the shared password.
- Selecting an option updates a compact summary and one primary action, `Continue as <role>`.
- The selected account's email is available as secondary context, not as an editable form field.
- Once authenticated, a compact `Viewing as <role>` control remains reachable without obscuring the application. Opening it exposes `Choose another account`.
- Choosing another account calls the existing logout endpoint and unmounts the entire application before the account catalogue is shown. No actor-scoped application state is reused.
- The chooser and account menu are keyboard accessible, have visible focus, expose selected/expanded state, and reflow to single-column controls on narrow screens.

## Document import experience

- When no imports exist, intake is open because importing is the only valid next action.
- When records exist, review content is primary and intake is collapsed behind `Import document`.
- The intake explicitly states supported formats and the 20 MiB limit before selection.
- A selected file shows trustworthy name, type and size metadata; selecting a file never uploads it automatically.
- Purpose remains required and source type remains explicit. Regulatory document selection remains a normal supported source classification.
- The primary button reports upload state and the durable created record becomes the selected import.
- Processing distinguishes `stored`, `processing`, `extracted`, and `stored only`. It never labels upload or extraction as compliance success.
- For PDF input, the current build retains the original and SHA-256 hash, then clearly reports that automated text extraction/review proposals are unavailable. This is an accepted stored-only outcome, not an upload failure.
- Existing exact quotes, source anchors, limitations, optimistic-concurrency proposal review and original-source reconstruction remain unchanged.

## Live workflow validation

Use public documents from official regulator domains as non-production demo samples. Validate each downloaded file as a real PDF, record its source URL and digest, upload through the same authenticated multipart endpoint used by the UI, poll the exact returned document ID, and verify:

1. a `201` durable receipt;
2. the original filename, size and SHA-256 are preserved;
3. the worker reaches a terminal state;
4. PDF terminal state is `UNSUPPORTED` extraction with `UNAVAILABLE` analysis and explicit stored-only limitations;
5. the record is visible in the actor-scoped list/detail UI;
6. another tenant cannot read the record;
7. no accepted proposal, requirement or compliance conclusion is created automatically.

Test records use filenames prefixed `demo-e2e-2026-08-12-` and purposes that identify the official source and state `workflow validation only`.

## Error handling

- Account login failures remain in the chooser without exposing protected application state.
- Account-switch logout failure leaves the current application mounted and reports a recoverable error.
- File and purpose validation are shown before upload and focusable by assistive technology.
- Network/upload errors preserve the selected file and purpose so the user can retry.
- Poll failures preserve the last durable receipt and retry later.
- Unsupported PDF extraction is rendered as a successful storage outcome with an unavailable-analysis explanation, not as a generic red error.

## Test strategy

- React tests cover compact role selection, hidden credential duplication, one-action login, account-menu disclosure, secure logout-before-switch, collapsed/open intake states, limit/help copy, retained retry inputs, durable receipt processing and stored-only PDF wording.
- Axe checks cover both the chooser and Imports workspace.
- Go tests retain size, tenant, digest, durable processing and unsupported-PDF contracts.
- TypeScript, Vitest, Go, deployment, build and diff checks run before push.
- Live official-PDF uploads and actor-scoped reads run after deployment; they do not imply legal review or compliance coverage.

## Acceptance criteria

- No username/password grid or duplicated credential form appears in the default demo sign-in path.
- A user can identify and enter a role with one selection and one primary action.
- A signed-in user can reach account switching in one control and the application is fully unmounted before the next identity is selected.
- Existing imports are the primary content; the intake form does not dominate by default.
- Upload requirements and state transitions are understandable without reading implementation terminology.
- At least two official regulatory PDFs complete durable upload and terminal stored-only processing on the live demo environment with correct hashes and no automatic governance conclusion.
- Default demo presentation and `?demo=0` production-style presentation behavior remain unchanged.

