# ClearSight V2.10 — Named responses, Documents and vendor onboarding

20 November 2026 · Interactive UI reference · Synthetic records

## Open the demo

Open **clearsight-v2-demo.html**. It starts at **Programs** and contains the application, styles and synthetic records in one file. It needs no installation, CDN or network connection. The **73 individual HTML entry points** use the adjacent `assets/` folder.

For a consistent browser origin, run `python3 -m http.server 8080` from this folder and open `http://localhost:8080/clearsight-v2-demo.html`. On Windows use `py -3`. File-origin persistence and downloads depend on browser policy. The demonstration snapshot is **13 November 2026**; the release date is separate.

## Assessment accountability

**Programs → Assessments** and **Forms → Assignments** now identify the person responsible for each response. A unit or supplier remains the assessment target; it no longer substitutes for a person's identity.

Each assignment records the named respondent, internal coordinator where applicable, assigned reviewer, deadline and assignment reason. Submitted responses retain the actual acting person, time, cycle, pinned questionnaire/requirement basis and answers. An external supplier respondent is distinct from the internal person capturing that response and its source reference. Two people assigned to the same target have separate drafts, submissions and histories.

Every assigned submission goes to review, including a form with no requirement mapping. The eligible program owner or assigned independent risk reviewer accepts or returns the whole response under their own current identity. Self-review and selecting someone else's name to sign a decision are blocked. Authority and scope are rechecked at decision time.

The response history shows submitted cycles, assignment/reviewer changes, returns, corrections and decisions. Corrections create another cycle; they do not overwrite the earlier submission. A pending replacement leaves prior accepted conclusions in place but does not count as current onboarding readiness. Acceptance of a questionnaire does not automatically mark its requirements Satisfied: the requirement decision remains a separate accountable step.

**Reports → Response accountability** adds named completion/review tables, due and review queues, history exports, charts and source links. New assurance board packs capture that context; old packs remain unchanged. Legacy records without a known submitter, time or review display that gap explicitly.

## One Documents workspace

**Documents** is the single file library. Its navigation covers All documents, Awaiting assessment review, Needs refresh and Assessment review history. A document workspace has one file/version history and an **Assessment reviews** tab for its contextual submissions and decisions. “Supporting evidence” is explanatory subtext for the document's role.

A contextual review links to the exact document, version, program, target and question. It distinguishes the person who submitted the document to that assessment from the original file uploader. The assigned eligible reviewer sees the source, validity, comments, earlier submissions and replacement links before deciding. Acceptance verifies the stored file bytes and rechecks the review after verification; missing, corrupt, archived or stale sources cannot be accepted through the new path.

File approval and assessment acceptance answer different questions and are labeled in the same workspace. The same file can support several contexts without duplication. Returned-document work remains actionable until its replacement is submitted. Existing Evidence URLs open the corresponding Documents view for compatibility; they do not expose a second library.

## Vendor onboarding from intake to activation

Use **Vendors → Onboard a vendor**. The four-step intake covers the legal supplier and named contact; the service boundary, exact location and accountable internal people; the assessment/document plan; and review before submission. Existing suppliers can receive a distinct service engagement without duplicate legal entities.

1. **Prepare and resume** the intake. A saved draft does not create a live service record.
2. **Submit for owner confirmation and independent risk review.** Returns preserve comments and the earlier intake revision.
3. **Launch due diligence.** Approval creates a service-specific program, real named baseline and questionnaire assignments, deadlines and document requests.
4. **Collect and review.** Open each response or requested document from the dossier. Documents use the canonical file store and exact-version approval process.
5. **Review activation.** The service remains in Due diligence until the approved plan, every required current accepted response, required current document approval and applicable requirement conclusions pass. The lifecycle reviewer rechecks that basis before activation.

The dossier shows progress, accountable people, blockers and source links. Internal coordinators receive external supplier obligations in Work/Inbox. There is no live supplier invitation or external messaging service in this local preview. Periodic review, suspension, resumption and verified exit remain included.

## Earlier completed work retained

| Area | Included behavior |
| --- | --- |
| Branch KRIs | Source-aligned question types, conditional context, monthly returns, review, exceptions and accountable actions. |
| Loss register | Four-step source-aligned capture; independent review; explicit reporting period; corrections; verified receipts and reversals; closure; draft-only CSV import. |
| Enterprise governance | Scoped roles/permissions, explicit denies, separation of duties, access review and escalation matrix. |
| Work and Inbox | Mentions, source events, deadlines, document refresh, review/correction queues and separate read/acknowledged/resolved states. |
| Controls, RCSA and risk | Scoped maintenance, durable drafts, sampled tests, independent conclusions and source/version checks. |
| Director/board reports | Practical charts, source tables, management actions, original-currency loss views and immutable captured packs. |

The KRI and loss mappings from the supplied operational-risk workbooks remain in `docs/`. Only source structure and classification vocabulary are distributed; the reference transaction narratives, customer names and amounts are not copied into synthetic demo records. Native XLSX import remains outside the package; the loss import accepts the original worksheet structure saved as CSV.

## Suggested review

1. Open a program's Assessments tab. Assign a named respondent and eligible reviewer; compare two respondents for the same target.
2. Switch to the respondent, submit a response and open its history. Switch to the assigned reviewer, return it with comments, then correct and resubmit without losing the earlier answers.
3. Open Documents, link one version to an assessment and inspect its Assessment reviews tab. Confirm the submitter and original file uploader are distinct fields.
4. Open Vendors, prepare an onboarding intake, complete owner/risk review and follow the resulting assignments/document requests. Inspect blockers before attempting activation.
5. Open Reports → Response accountability and capture an assurance board pack. Later source changes must not alter the captured pack.

## Build, verification and delivery boundary

Run `python3 build.py`. The Node suites listed in `docs/TEST_REPORT.md` exercise actual-source models, commands, routes and actions. They substitute the browser shell and do not validate real layout, file persistence, downloads or printing. Browser execution remained blocked by the available environment; no V2.10 screenshot or browser-layout claim is made.

Run `python3 verify_reference.py` to verify the untouched release. Rebuilding or rerunning tests can change manifest hashes.

This updates the **interactive design package**, not a production deployment. Browser state and identity switching demonstrate the UI contract. Production needs authenticated identity and scoped retrieval, server-side authorization, transaction/concurrency controls, durable storage, real notification delivery and scheduled escalation. The demo sends no real messages.

Current details: `docs/ASSESSMENT_TRACKING_V210.md`, `docs/DOCUMENTS_V210.md`, `docs/VENDOR_ONBOARDING_V210.md`, `docs/V210_REVIEW_NOTES.md`. V2.9 handoff documents and earlier screenshots are retained as historical material.
