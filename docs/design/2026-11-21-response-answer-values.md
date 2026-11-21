# Response answer values

Response review must show the stored answer rather than infer a result. The response API returns a field type and textual values, but does not provide the configured currency in this projection. Numeric text therefore remains unchanged: no rounding, numeric conversion or assumed currency. A valid percentage gains its field's `%` unit. For checkbox and attestation fields, recognized `yes`/`true` and `no`/`false` values display as confirmed or not confirmed; unrecognized text remains visible and a missing answer stays missing. Submitted artifact IDs indicate a signature, photo, or a document count; they do not identify the signer or establish document review. Structured document fields remain separately visible. Existing multiline whitespace is retained.

## Evidence and limits

The v2.15 reference includes `clearsight-v2.15/assessment-response-review.html`. This is a value-rendering change, not an approval-screen cutover. The prior response render is retained under `docs/evidence/2026-11-21-review-revoked/` but uses different data. The new synthetic `response-values` fixture is retained under `docs/evidence/2026-11-21-response-values/`; reviewer `/root/ci_migration` inspected 1440 and 390 pixel captures. Browser checks passed at 1440, 768 and 390 pixels, with no horizontal overflow and no axe violations with color contrast excluded. The fixture's document count is consistent with its document list.

Unit tests cover exact numeric precision, missing and unrecognized confirmations, multiline text and artifact counts. This evidence does not establish source-to-record verification with a live backend, currency metadata, identity of a signer, contrast, dark mode, keyboard review, or complete v2.15 screen acceptance. No design tokens were added.
