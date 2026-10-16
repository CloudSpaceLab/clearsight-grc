# #462 — Reference Program / Form / Issue receipt gate

This is an automated **non-production persisted-data acceptance tranche**, not hosted actor UAT. It validates the existing installer and uses no new workflow, fixture user, Program, Form, scoring policy, or report implementation.

## CI implementation

The deployment migration job runs `cmd/seed-bank-reference -demo-journeys-only` twice against the same isolated PostgreSQL database. The existing server installer emits both receipts. `deploy/scripts/verify_reference_acceptance.py` validates them, and `deploy/tests/test_reference_acceptance.py` checks deliberate identity/replay failures.

| Verified source | Passing expectation |
| --- | --- |
| Program and three canonical Matters | Stable subject/record IDs across installation and replay |
| Program response Form revision | Existing active version, three distinct current scored response revisions linked to the same Program |
| Vendor Forms | Five explicit sample states, stable across replay; no draft mistaken for submitted |
| Response policy | Exactly one Issue despite a distinct poor-response amendment; stable active policy/version and Issue ID on rerun |
| Oversight | Nonempty population and authoritative Matter/event high-water in both receipts |
| Installation mode | Explicit non-production seed mode; no implicit sample installation |

The verifier prints counts and verification flags, **not** recipient addresses, tokens, answers, private OpsRisk source, or evidence contents. It fails closed if record/source identity changes.

## Not yet accepted

- Admin authorization and customer-specific configuration of a new Program.
- Real external recipient delivery/identity assurance and contributor response under a different actor.
- Bank reviewer evidence judgement, routed authorizer and independent outcome-check receipt.
- Authorized generated report, actual download, and report-to-source row parity.
- Hosted bank/telco tenant-isolation, service permissions, private OpsRisk import or enterprise-cardinality proof.

Those require separate actor-authenticated API/browser and deployment receipts. Do not mark #462 T1/T4/T6 complete or infer a Program compliance percentage from this demo fixture.
