# External capture invitation entry

The v2.15 response reference is `clearsight-v2.15/form-response.html` (SHA-256 `59A8C0619CD7667B9F15B204441143E3978141DFF65B5777745A41483D6AFE82`). This focused tranche retains the current external capture composition; the reference does not establish the backend access or identity contract.

A direct email-verified invitation sends a code only after the recipient opens the request and the server returns exactly one recipient. A delivery failure retains that recipient and offers a manual retry. Ambiguous direct recipients and duplicate invitation parameters fail closed. Invitation fragments are removed before capture opens. Session, request and distribution identifiers must match the returned workspace during initial access and conflict recovery; unsupported question types continue to block submission.

Synthetic code and failed-send recovery states at 1440, 768 and 390 pixels are retained in `docs/evidence/2026-11-21-v215-capture-entry/`. The browser check verifies fragment removal, preserved query, retry, masked recipient, lack of persistent session credentials and horizontal overflow. The 390 and 768 failed-send renders and 1440 code render were inspected. Fixtures do not establish real email delivery, server authorization, backend scope, screen parity, dark-theme behavior or production acceptance.

The legacy `ExternalCaptureApp.test.tsx` remains in place: its full entry, terminal OTP and conflict assertions have not all been migrated. Source-bound prefill and previous-response provenance remain unsupported and must not be enabled without their governing contracts.
