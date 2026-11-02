# Approved ClearSight UI designs — 17 October 2026

Source: [Complete designs (Google Drive)](https://drive.google.com/file/d/1y1Ij8zyfZKUKfcxp68T9FmJHabyWREZ_/view?usp=drivesdk)

Original archive: `complete designs.zip` (78,899,802 bytes; SHA-256 `65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5`).

The archive contains **18 nested ZIP packages**, including framework, program, assessment, vendor, operational risk, privacy, reporting, work/approvals, controls/policies, enterprise setup, search, notifications, audit, people/access, automation, integrations, contextual Copilot and the unified design system. The source package is intentionally linked instead of committing 79 MB of redundant binary previews to the application history. It is not a build dependency.

## Framework source files

The nested `clearsight-framework-designs.zip` contains:

| File | SHA-256 |
| --- | --- |
| `framework-library.html` | `193d13b0cf87381f20b7d84c3d4c50c80b0e064274a623fa98257506df8284bf` |
| `framework-adoption.html` | `8cad235bcccc1131964f5e37c237b3034075fb8130e45d7bf667111bcfc963ed` |
| `framework-library-4k.png` | `e29f8089e678b4fb0d4a9153b1a79a18abb49c8750963333cbd7a363fbfb9bf8` |
| `framework-adoption-4k.png` | `29d10977569d645f713b994c633813ade8012f0ab9021d82dbd0c8b8aa1dc23e` |
| `framework-adoption-mobile.png` | `fee109b931c72479de4066aa8177629b7ce0513cc5ded1f1d57ed09e48a76e2c` |

The `clearsight-program-designs.zip` package also includes `program-workspace-preview.html` and responsive previews. The `clearsight-unified-design-system-v1.zip` package contains `clearsight-design-system/clearsight-design-system-spec.md` plus HTML, CSS, JS and visual references. Its spec explicitly names #462 and is the design contract for the cutover.

## Implementation mapping

- **Application shell:** existing `App.tsx`, routing, enterprise shell, scope and date controls. Do not fork navigation.
- **Framework catalog/adoption:** existing #437 catalog and #433 Program scoring contracts. Edition provenance, source rights and requirement mapping precede any percentage.
- **Program and forms:** reuse existing Program components and Forms workflows. Adoption must preserve distinct answer, assessment and approval states.
- **Common components:** `web/src/design-system/` and `web/src/components/ui/`; prefer shared Button, SearchField, FilterBar, Tabs, MetricCard, DataTable, FocusedSheet and status components. Do not paste independent prototype CSS or static fake data into production.
- **Accessibility and cutover:** verified desktop/mobile, light/dark, scope, role, empty/partial/stale, and reduced-motion states. Follow #462 acceptance and staged PRs.

## Source import

The source package is currently **referenced, not copied into Git**. To create a tracked source snapshot, extract selected HTML/spec files from the nested packages into `docs/design/references/clearsight-2026-10/`, preserving their exact bytes and hashes; upload screenshots separately as review artifacts or Git LFS, not into the deployed web bundle. Never ship the standalone static preview as a production route.

## Acceptance

Reconcile the approved design against real administrator, contributor, reviewer and authorizer journeys. Verify imported framework → Program adoption → Form submission → review → approval → report export against source records and role/tenant isolation. Do not mark the UI cutover complete solely from visual previews.

Related: [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462), [#437](https://github.com/CloudSpaceLab/clearsight-grc/issues/437), [#433](https://github.com/CloudSpaceLab/clearsight-grc/issues/433).
