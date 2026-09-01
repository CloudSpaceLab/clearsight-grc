# UI component adoption

This matrix records where ClearSight's shared UI contracts are actually enforced. It prevents a completed foundation or one migrated screen from being reported as product-wide visual consistency.

**Legend:** **M** = migrated and enforced in the migration manifest; **S** = shared shell/navigation contract used, but the workspace body is not migrated; **—** = not migrated in this tranche. A row is complete only when every family needed by that workspace is migrated and its full-host evidence passes.

| Workspace or surface | Actions | Fields | Selection | Navigation | Overlays | Feedback | Surfaces | Data |
| --- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| UI component gallery | M | M | M | M | M | M | M | M |
| Forms shell and peer-view navigation | — | — | — | M | — | — | — | — |
| Forms · Sent forms | M | M | M | M | M | M | M | M |
| Forms · Templates/library | M | M | M | M | M | M | M | M |
| Forms · Builder/editor/review | — | — | — | S | — | — | — | — |
| Forms · Responses | — | — | — | S | — | — | — | — |
| Forms · Imports | — | — | — | S | — | — | — | — |
| Forms · Communications | — | — | — | S | — | — | — | — |
| Forms · Policies | M | M | M | M | M | M | M | M |
| External Forms capture and invitation access | — | — | — | — | — | — | — | — |
| Today | — | — | — | — | — | — | — | — |
| Oversight | M | — | — | M | — | M | — | M |
| Programs and dedicated Program record | — | — | — | — | — | — | — | — |
| Vendors and vendor relationship record | — | — | — | — | — | — | — | — |
| Work · Issues and changes | — | — | — | — | — | — | — | — |
| Work · Evidence | — | — | — | — | — | — | — | — |
| Imports · regulatory document comparison | — | — | — | — | — | — | — | — |
| Explore · reference journeys | — | — | — | — | — | — | — | — |
| Configure · policy, routing and integrity | — | — | — | — | — | — | — | — |
| Shared shell, primary/mobile navigation and account context | — | — | — | — | — | — | — | — |

## Implemented boundary

Tranche 1 implements the three-layer token architecture, public component contracts, component gallery, Forms peer-view navigation and the complete Sent forms workspace migration. The Sent forms boundary includes its actions, filters, themed selection popup, labelled empty result, populated and paginated data, focused detail sheet, lifecycle feedback and narrow-screen stacked records.

Tranche 2 begins the Templates/library migration at its highest-frequency record surface. The template table now uses the shared data, status, checkbox-selection and action contracts; template details use the shared focused sheet, actions and status feedback; and workspace notices and empty results use the shared feedback contracts. The component foundation adds one labelled `CheckboxField` contract and numeric constraints to `TextField`. These exact files are enforced by the migration manifest and exercised in the component gallery and Forms library state fixtures.

Tranche 3 adds a shared `ActionCard` for consequential routes and migrates the new-form launcher to shared action cards, buttons, record cards, empty state and focused sheet.

Tranche 4 completes the Templates/library boundary and removes the most visible Forms control inconsistencies. Template search, removable filters, typed filter selection, bounded advanced expressions and lifecycle scopes use the shared search, filter, popover, field, action, sheet and scope contracts. The creation launcher uses the centered `FocusedDialog`; every Forms select now uses `SelectField`; Responses empty/error/action states and Communications lifecycle/test-send actions use shared feedback, field and button contracts. The remaining Builder inputs, Imports, list selectors and rich-text toolbars are still migration work. Every non-Forms workspace also remains outside this tranche.

The governed-work handoff slice applies the shared contracts to the highest-risk interactions in the otherwise unmigrated Work and vendor-request surfaces: issue and Action reassignment, vendor-request composition, response review/sign/transmit/acknowledgement, issue authorization, date entry and list overlays. These files still contain older secondary forms, so the Work and Vendors rows remain unmarked until those complete surfaces are migrated and added to the executable manifest. This partial boundary must not be described as a completed workspace migration.

The Oversight slice uses shared buttons, automatic keyboard tabs, responsive data tables and scoped empty states. Its local metric, intervention and analysis surfaces remain feature-owned because they encode the oversight information hierarchy rather than a reusable generic card family. Both its React control boundary and feature CSS are enforced by the migration manifest.

The scoring-policy slice adds the Policies peer workspace as a fully enforced shared-control boundary and migrates the advanced-scoring portion of the Builder inspector. Policy lifecycle actions, focused creation, typed effective dates, list selection, feedback and bounded data states are registered in the executable manifest. The Builder row remains incomplete because other editor controls still sit outside this slice; this change must not be described as a complete Builder migration.

## Work left after this tranche

The next migration slice should finish the remaining raw controls in Matter details, Actions, decisions, outcome checks and vendor-response recovery, then add those complete files to the manifest. Forms Builder, Imports, remaining list selectors and external capture continue as separately evidenced slices.

Product-wide adoption needs a separate sequenced plan. Start with shared shell/navigation, then migrate Today, Programs, Vendors, Work, regulatory Imports, Explore and Configure according to task frequency and operational risk. Each slice must preserve domain and authority behavior, add its manifest entries and state fixtures, inspect full-host renders, repair the highest-impact defect and update this table. Until those rows are migrated, describe the result as “UI foundations, Sent forms and the Templates workspace migrated,” never “ClearSight is fully standardized.”
