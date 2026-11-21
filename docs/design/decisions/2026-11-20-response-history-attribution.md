# Response history and attribution — partial integration

The v2.10 assessment history and review references call for named people and immutable response cycles. The Program response list now offers the existing authorized, paginated historical-revision query. The review shows the actual submission actor returned by the assessment service; the HTTP layer resolves an optional display name only after authorization and only within the exact tenant/entity. Missing or unresolvable identities remain explicitly unknown. Historical vendor answer views use the actual currency state rather than claiming current status. Vendor response document labels refer to documents rather than a separate evidence library.

These changes use existing responsive tables, checkbox, sheet, status and facts styles; no new tokens. Required fixtures cover recorded and absent submitters, exact scoped name lookup, current and historical revisions, and denied/self-review behavior. History is a list of authorized response revisions, not a complete assignment, routing or review-event timeline.

This is partial V2 layout integration. It does not provide v2.10's named-assignment lifecycle, universal whole-response acceptance/return, response-review inbox projection, canonical document consolidation, or onboarding dossier. The existing assessment model can say review is not required when a form has no review rubric. It must not be represented as the requested universal review. The existing vendor creation form creates a relationship, not the v2.10 onboarding workflow. API-backed question work remains as described in `docs/superpowers/plans/2026-11-20-api-question-contract-follow-up.md`.

Layout and production acceptance require authorized real-backend state, same-screen source/runtime comparison, responsive renders, keyboard/touch and independent review evidence. No acceptance is asserted by this change.

## 20 November follow-up: opening a response revision

The Forms response History list can open an earlier immutable revision through the existing authorized response reader. The returned response and revision must match the selected history item, distribution, subject and form version; a retry retains those checks. The opened response keeps its own answers, documents and historical status. The local `forms-response-history` fixture includes both exact responses, and the evidence scenario opens the earlier revision.

Reference: `clearsight-v2.10/assessment-history.html` SHA-256 `b3763d60908dc37d2578beeff01a9586e37f1bda3595413c70ca06ecf4919760` and `clearsight-v2.10/assessment-response-review.html` SHA-256 `779ebaea9cd1f74e774d94cb1eb00eaf09ece8e8d66913483f35104e8f4ae37d`. This is a functional adaptation within the existing review sheet, not a v2.10 page-layout cutover. Fixture renders were inspected at 1440, 768 and 390 pixels; they are not evidence of a same-screen source comparison or production authority.

## 20 November follow-up: submitted-by lists and document review labels

The bounded Program and Forms response lists show the recorded submission actor. Authorized PostgreSQL and memory reads obtain the actor from the submission attached to the response revision; the HTTP layer resolves optional names within the same tenant and legal entity. Missing identities remain unknown and an unavailable directory does not change the recorded attribution. This identifies the actual submitter, not a named assignment or an external respondent captured by a coordinator.

The existing submitted-document table shows its recorded assessment-review status separately from file availability, and the detail uses the same labels. An absent review remains “No review recorded.” The underlying endpoint only exposes its existing submitted-file contexts and vendor-assessment reviews; it does not establish the v2.10 canonical document library, universal assessment-review queue, or file-approval history.

## 20 November follow-up: sent form recipients

The existing sent-form detail now lists the authorized distribution's stored recipient labels, respondent/copy role and delivery or submission state. Missing names remain unavailable; a completed response is not presented as independently reviewed. The v2.10 Forms reference is `clearsight-v2.10/forms.html` SHA-256 `8d8a5969b3ceb5bb9cfc09d5ec023b35e0164ae488ebb93143285959925b237c`. Fixture renders at 1440, 768 and 390 pixels were inspected; this remains a partial integration, not the v2.10 assignment page. Existing distributions can share a response workspace and are not the separate named assignments or universal reviewer routing required by the reference.

## Link response recipient provenance

New link-submitted response revisions retain the exact authorized recipient route and its safe label as a snapshot, distinct from the actual submitter. Assessment detail displays these as separate facts; link possession or email verification does not establish the identity of the person who submitted. Existing revisions do not acquire invented recipients. The standard Forms response-history fixture covers a recorded external recipient and an unknown submitter. The v2.10 response-review layout remains uncutover; this reuses the existing facts component and does not add tokens.
