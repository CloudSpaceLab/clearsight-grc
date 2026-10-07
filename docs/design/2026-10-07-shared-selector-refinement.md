# Shared selector refinement

## Decision and baseline

Bank operators repeatedly narrow records by owner, rating, vendor and workflow state. The shared selector already provided a bounded, keyboard-operable option list, but its text-glyph disclosure was visually weak. Vendor exception filters also bypassed that control with native browser selects, producing a visibly different affordance.

The retained approach strengthens the existing `SelectField`: a compact SVG disclosure indicator, hover/pressed/open feedback and a rotated open state. The indicator does not create a second action; the complete field remains one 44px minimum target. Vendor exception status, vendor, owner and source-rating filters adopt the shared control. No new control family, token, density mode or workflow state is introduced.

## States and responsive behavior

- Default, hover, pressed and open states retain the existing field boundary and semantic colors.
- Focus remains visible; keyboard Arrow, Enter, Escape, Tab and option selection retain the established React Aria behavior.
- Disabled, invalid, placeholder and selected states remain on the shared contract.
- The indicator uses motion only for the open/closed state and stops when reduced motion is requested.
- On narrow vendor views, the status selector replaces the desktop status buttons; the three facet selectors retain their existing three-column responsive arrangement.

## Proof

- `SelectField.test.tsx` verifies the SVG disclosure indicator and open state.
- `VendorPortfolio.test.tsx` verifies the migrated vendor filters and their filtering behavior.
- Static gallery renders were inspected for closed and open selector states at a narrow viewport. The first visual check confirmed the selector's open menu was legible and the indicator state remained clear; no repair was required.
