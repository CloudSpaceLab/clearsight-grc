# Program follow-up release proof

Decision: group internal assessment failures separately from collection-source failures and open issues. Show affected checks on expansion, retain calculation/version context, and navigate directly to the relevant tab. Keep the primary action beside the status heading on desktop; stack it on narrow screens. Do not treat an expired internal assessment as proof that a vendor document has expired.

The submitted-response table combines concern and score into one assessment result. Scoring coverage remains in response detail; it is not document currency. Existing current-response pagination is unchanged.

Baseline: customer-supplied screenshots dated 25 September 2026 show repeated expiry paragraphs, a large read-only action box, and separate score/concern/coverage columns. Render fixtures use named assessment contracts with expired validity and two open issues. The static response population is explicitly sample data.

Rendered evidence:

- [Desktop overview](screenshots/program-follow-up/desktop-overview.png)
- [Mobile overview](screenshots/program-follow-up/mobile-overview.png)
- [Desktop responses](screenshots/program-follow-up/desktop-responses.png)
- [Mobile responses](screenshots/program-follow-up/mobile-responses.png)

Verification: 107 focused web checks passed; typecheck and evidence build passed; continuity, evidence and workflow Go packages passed; PostgreSQL-tagged notification checks passed. Both 1440px and 390px renders have no page-level horizontal overflow. Clicking Review assessments reaches Evidence & results. Copy-quality check passed again after the layout refinement.

Email receipt migration adds the comment-mention notification kind without removing existing receipts. Actual external email delivery requires the configured mail transport; no external email was sent during verification.

Broader workflow limitations are recorded in the implementation plan's release scope. This change does not remove synthetic scoring acceptance records from the demo database.
