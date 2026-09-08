# Forms section resumption

Status: implemented bounded IGX-00 slice under #200. Exact-head review and release receipts are tracked in [PR #203](https://github.com/CloudSpaceLab/clearsight-grc/pull/203) and #147; broader document acceptance remains open.

## Decision

Keep the existing Forms peer tabs and Finder-style document browser. Encode only the selected peer section in a validated `section` hash-query parameter. Reload, direct links and browser Back/Forward should open the same section. Templates remains the default, including for unrecognized section values. Existing `#forms/<template-id>` links and template filter URLs retain their meaning.

The user approved connected journeys and safe navigation recovery in #200 and asked to continue. This is a bounded IGX-00 correction, not a new document workflow or visual redesign. No new mockup is required or requested. Alternatives rejected: a second router or persisted workspace store would add unnecessary scope; treating section names as path IDs would conflict with existing template links.

## Interaction and safety

- Directly choosing a different tab adds one history entry. Choosing the active tab does not add another. Back/Forward changes the selected tab and its existing labelled panel.
- Main desktop/mobile Forms navigation intentionally opens the Templates root, including when Forms is already mounted. Its URL and selected section must agree before and after reload.
- Preserve the current template target and template query while changing peer sections. Existing template-filter writes keep their established replace behavior.
- Returning to Documents mounts its existing reader and re-fetches through the existing scoped API. Templates preserves a matching pending read and its existing revalidation behavior. The URL is navigation intent, never authorization or evidence of current access.
- Only an allowlisted section slug is newly persisted. Do not add file names, document text, respondent details, document queries, credentials, access tokens or draft contents to URLs or browser storage.
- Actual tab changes clear the same transient editor/creation/AI/error/notice state as the existing click handler. Repeated history events for the same tab must not erase an active editor.
- Keep current shared tab keyboard behavior, accessible selected state, panel relationship, focus styling, responsive layout and loading/error/empty content. No new tokens, copy or dependencies.

## Required proof

Unit/component coverage: every valid section, absent/unknown values, direct Documents mount, history events, no duplicate history, transient-state behavior, exact template target/filter compatibility. Browser proof: select Documents, reload, choose another section, Back and Forward; confirm fresh document reads and the selected tab/panel relationship. Check light/dark desktop, 390px and 320px with no overflow or blocked primary actions. Retain a before-state receipt demonstrating the reset to Templates.

## Explicitly remaining

Document filter/page/occurrence/preview resumption, vendor document return paths, document review/replacement/disposition, validation rulesets and the wider IGX-00/08/11 acceptance remain open. This slice does not prove recipient, bank-user timing or production custody outcomes.
