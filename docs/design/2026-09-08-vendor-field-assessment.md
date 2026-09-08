# Vendor forms and bank assessment decision brief

Approved on 8 September 2026 after synchronizing the checkout with main `60a6a6065b7ca6de7f52b66dd24c36fb19fc0904`.

The register answers which vendor still owes a form, which submission awaits bank review and which current assessed submission has adverse results. Counts describe the authorized records checked at the displayed time. The register's form-work filter explicitly covers loaded relationships; each vendor's response filters run in the API before pagination. No enterprise-wide total is inferred from a page.

Reuse the existing form importer, immutable template revisions, scoring profile editor, distribution composer, recipient access, evidence browser and policy lifecycle. A field adds one of four modes: unscored, automatic rules, bank rubric, or automatic rules followed by bank review. Manual decisions reference the exact submitted field and do not rewrite the respondent's answer. Combined assessment preserves critical automatic effects and cannot silently improve an automatic contribution.

The vendor record presents form requests and response review directly. Bulk setup assigns one recipient to each selected service; the confirmation lists every vendor/contact/form/deadline. Saved per-target creation receipts make retries recover existing requests. A prepared request with failed access setup is visibly saved and retryable. Created, delivered, answered, submitted, assessed and accepted remain separate facts.

The bank review sheet places respondent answer and bank judgement together, includes a filter for pending review and poor results, and explains contributions from the pinned scoring profile. Shared conditions identify every referenced field instead of assigning a cross-field result to an arbitrary question. Configured rule names and purposes accompany specialist references.

Responsive replacement: vendor rows become stacked cards; actions wrap; answer and judgement columns stack within narrow sheets; no fixed desktop widths or hidden status labels. Existing semantic color, spacing, button and sheet tokens are reused. Keyboard focus remains in the active sheet and returns to its trigger. Empty and unavailable states preserve request/retry actions without fabricated zeros.

Required fixtures: incomplete required fields, saved-but-unsubmitted answers, awaiting bank review, completed poor assessment, historical response, absent scoring, partial dispatch/retry, empty forms, unavailable forms, mixed field authoring and inaccessible review. Evidence fixtures live outside the production import graph. Baselines are preserved under `.codex-tmp/vendor*-before*.png`; the acceptance record links the final render manifest.

Final visual evidence uses 72 light/dark cases at 1440, 390 and 320 CSS pixels, with four tablet and four 200% reflow-equivalent checks. The compact register uses inline summary links with the existing button-height token scoped to the summary; each row keeps its checked time and qualified concern. Partly replaced responses state that the displayed score covers the earlier full response and that remaining fields require review. Separate response history preserves current/history labels and opens the selected revision without offering changes to a historical judgement. The final acceptance record contains the manifests and the exact reflow method; all checked cases have no horizontal overflow or axe violations.

Inline summary links use the existing compact action-link tokens with a minimum 24px target, leaving Request form as the dominant vendor action. Partly replaced responses retain their original assessed concern with a visible qualification; history shows the source versions without recalculating a favourable result from an incomplete replacement.
