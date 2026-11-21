# Documents consolidation in V2.10

## One public concept and one file store

The user-facing library is **Documents**. Its top navigation offers All documents, Awaiting assessment review, Needs refresh and Assessment review history. Canonical document workspaces have Overview, Versions, Linked records, Approvals, Assessment reviews and Activity. “Supporting evidence” appears as explanatory context for a document used in an assessment.

The internal legacy `S.evidence` collection remains a set of contextual review records referencing canonical document versions. It is not another file store, and existing IDs/hashes/history are not renamed merely to change UI copy. Old `evidence` URLs route to the Documents review view. Old `evidence-review?id=…` URLs open the precise document/version/context.

## Contextual attribution and review

A context records program, target, question, observed/valid-through dates, actual submitting actor/time and assigned eligible reviewer. The original file uploader remains a separate fact. Legacy records without captured submitter/time display that absence; migration does not invent attribution.

The Assessment reviews tab shows the exact file version, current context state, assigned reviewer, review comments, previous/replacement submissions and named submission/decision history. A context status such as Accepted is explicitly labeled Assessment review; the general file approval state remains separately labeled. This avoids interpreting a file's generic Draft status as a rejection of an accepted contextual review.

A creation command validates scope, target/question context and reviewer eligibility before creating the review record. Reusing an existing context does not overwrite its submitting identity. New contexts can share the same file/version without copying it.

## Decision integrity

`d210Review(id, data, expectedVersion)` is asynchronous. It permits Accept or Request changes only for the assigned currently eligible program owner or independent reviewer. The actual actor must match the decision identity and must not be the contextual submitter. The original uploader alone does not determine who prepared the contextual submission; legacy records without stronger identity use the available owner conservatively.

Acceptance checks source availability, current dates and program context, then calls `docVerifiedBytes`. After the await it rechecks actual actor, record stability, permission, scope, source availability and version before writing the decision. Corrupt bytes, unavailable files, archived files, stale scope or changed actor leave the record and receipts unchanged. Request changes remains available when source correction is needed.

The original context validator is used when checking an existing submission for acceptance. The creation-time reviewer-capacity check must not treat the deciding program owner as a new submitter and exclude the sole eligible reviewer. A regression covers that case.

A decision writes one shared receipt, actor/time/comments and a history entry. Acceptance of a replacement marks earlier contexts Superseded while preserving their original review and replacement lineage. Returned notifications and Work items point to the same canonical document context. They remain open until a replacement is submitted.

## Integration surface

The new module is `assets/documents-unified.js` with companion CSS, loaded after assessment tracking and vendor onboarding. Existing `documents.js` exposes the assessment-review tab and top navigation; `flow-polish.js` collects the reviewer during linking/refresh. Inbox uses the actual submitter and assigned reviewer.

`d210ReviewRoute` resolves the exact canonical route. `d210Reviewers` and `d210ReviewEligibility` reuse assessment reviewer policy. The legacy `jEvidenceDecision` adapter now returns a promise and must be awaited. Browser submit handling awaits that promise before rerendering.

The document approval workflow and contextual assessment review remain distinct decisions in the same workspace. Vendor requests use the canonical exact-version file approval mechanism; they do not create another document repository or pretend that attachment is approval.

## Verification

`node tests/documents_v210_checks.js` covers terminology/navigation, canonical version reuse, source routes, actual submitter/uploader distinction, eligibility/assignment/self-review, byte integrity before mutation, returned Work/Inbox ownership and missing legacy provenance. The current flow regression adds refresh/resubmission and supersession history; the crossflow suite adds the sole eligible program-owner regression and assessment/vendor integration. Tests execute actual source with a substituted browser shell; browser file-upload interaction and visual layout remain unverified.
