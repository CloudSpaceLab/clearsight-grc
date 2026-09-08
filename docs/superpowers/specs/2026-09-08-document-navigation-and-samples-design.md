# Document navigation and realistic submitted samples

## Approved scope

The user approved this scope on 8 September 2026, following the findings and acceptance checklist in [master #200](https://github.com/CloudSpaceLab/clearsight-grc/issues/200#issuecomment-5586374921). This is an addition to the approved vendor activation handoff/recovery work, not a replacement for it. No more mockups or new document module.

## Decisions

Keep desktop Forms peer tabs and the Finder-style file-type sidebar. At narrow widths replace each with a compact, visibly labelled selector so seven sections and six file types do not consume multiple rows or clip labels. Reuse existing SelectField and Tabs contracts. Preserve the selected section, query and mounted content across viewport changes; viewport changes must not discard an editor draft or document selection. Keep normal keyboard navigation, focus, accessible panel naming and URL reload/Back/Forward behavior. Document filenames use the full card width on narrow screens, with metadata underneath and Preview still reachable. Do not restyle shared component internals from feature CSS.

Use a small connected fictional sample pack through the existing non-production installer and submission/artifact paths. Screenshot-only metadata cannot prove preview or persisted workflow behavior; a large synthetic population adds maintenance without closing this gap. Retain the existing scoring fixtures. Add meaningful vendor service, contact roles, typed answers, last-submitted dates, file issue/expiry dates and immutable revision links. Real readable sample files have actual media type, byte size and checksum. Every record/artifact is labelled sample, with no genuine confidential data, counterfeit certification or invented signature.

Sample scenarios cover current, approaching expiry, expired, missing evidence, contradictory vendor/date, previous/replacement revision and pending inspection. Stored status must match actual completed operations: do not mark bytes inspected or evidence accepted just to make a screen look complete. Signatures are required only by a form signature field. Acceptance, submission and verified outcome remain distinct.

The same stored sample occurrence must be discoverable from response detail, Forms Documents and its vendor relationship. Production installation is refused. Repeat installation is idempotent, partial installation recoverable, and genuine submissions untouched. No mail to real recipients and no external AI processing.

## Delivery boundaries

1. Responsive navigation and filename readability, independently testable without data mutation.
2. Vendor activation stale-result recovery and access-filtered exact-record handoffs, preserving material authority checks.
3. Connected sample installer and actual protected-byte browser acceptance.

Each has its own implementation tasks and verification receipt. Document disposition/replacement commands and validation rulesets remain separate work in the existing child issues, with scope retained in closed #200; sample scenarios are acceptance inputs, not claims that those capabilities exist.

## Required proof

Preserve before screenshots. Test desktop and 720/390/320px, light/dark, 200% reflow, keyboard, slow/error/empty states and resize while editing/previewing. Run copy-quality, affected workflows, UI contracts, runtime fixture isolation, full web/Go and rendered review gates. Inspect renders and repair the highest-impact finding. For samples additionally test production refusal, exact scope/access denial, repeat/partial runs, response/artifact membership, integrity and actual preview/download. Merge/deploy only green exact-head code, verify the hosted revision, and record remaining sample, vendor and release limitations in #138/#139/#147. Keep #200 closed as instructed.

## Operator amendment: demo scan simulation

The user initially authorized configuring local ClamAV, then chose a decorated demo scan experience after the host capacity check. On 8 September the shared Rocky Linux host had approximately 1.9 GiB available RAM, no swap and 11 GiB free disk. ClamAV was not installed or configured; no server services were changed. Live antivirus enablement is deferred, not completed.

The demo must say **Demo scan** / **Demo check complete**, with **No antivirus scan was performed**. It must never say that a real antivirus product scanned the file. Do not create a CLEAN receipt, change scan jobs or mark artifacts AVAILABLE to decorate a demo.

For usable fictional-file previews, use a narrow demo-only allowlist of shipped sample bytes, identified by actual digest, size and media type. Reuse the normal protected occurrence/content routes and full stored-byte integrity verification; do not serve a different asset while claiming it is the submitted original. Only an unscanned known sample can use this exception, and its pending antivirus status remains stored. Quarantined/deleted/changed/unknown files remain blocked. The exception is disabled outside demo mode, and existing production refusal of demo mode remains mandatory. Review acceptance still requires its existing real inspection and authority gates; previewing a fictional sample does not change them.

No ClamAV package, scanner process, synthetic scanner receipt, database status migration or artificial waiting timer is needed for this simulated presentation. Add automated tests for arbitrary-byte rejection, matching-name/different-content rejection, quarantine, scope revocation and non-demo/production refusal. This amendment replaces only live scanner installation, not realistic sample data, navigation or genuine-upload security requirements.
