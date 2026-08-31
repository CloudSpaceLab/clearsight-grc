# UI component adoption

This matrix records where ClearSight's shared UI contracts are actually enforced. It prevents a completed foundation or one migrated screen from being reported as product-wide visual consistency.

**Legend:** **M** = migrated and enforced in the migration manifest; **S** = shared shell/navigation contract used, but the workspace body is not migrated; **—** = not migrated in this tranche. A row is complete only when every family needed by that workspace is migrated and its full-host evidence passes.

| Workspace or surface | Actions | Fields | Selection | Navigation | Overlays | Feedback | Surfaces | Data |
| --- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| UI component gallery | M | M | M | M | M | M | M | M |
| Forms shell and peer-view navigation | — | — | — | M | — | — | — | — |
| Forms · Sent forms | M | M | M | M | M | M | M | M |
| Forms · Templates/library | — | — | M | S | — | M | — | M |
| Forms · Builder/editor/review | — | — | — | S | — | — | — | — |
| Forms · Responses | — | — | — | S | — | — | — | — |
| Forms · Imports | — | — | — | S | — | — | — | — |
| Forms · Communications | — | — | — | S | — | — | — | — |
| External Forms capture and invitation access | — | — | — | — | — | — | — | — |
| Today | — | — | — | — | — | — | — | — |
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

The next creation slice adds a shared `ActionCard` for consequential routes and migrates the new-form launcher to shared action cards, buttons, record cards, empty state and focused sheet. Search, filter chips, typed filter popovers, advanced expressions and status scopes still contain legacy controls; the Templates/library row therefore remains partial rather than promoting the whole Actions, Fields, Overlays or Surfaces families.

Template search, filter selection, saved filter composition, status scopes and the new-form launcher remain legacy controls, so Actions, Fields, Overlays and Surfaces are intentionally not marked migrated for the Templates/library row. The builder, Responses, Imports and Communications remain visually functional legacy surfaces. Builder usability fixes already present—mobile action visibility, keyboard reordering and pointer reordering—are retained, but they do not constitute component-system migration. Every non-Forms workspace also remains outside this tranche.

## Work left after this tranche

The next migration slice should complete template search, filters, status scopes and the new-form launcher before moving into the builder. The builder remains the largest concentration of native selects and inconsistent component anatomy. Responses, Forms Imports, Communications and external capture follow as separately evidenced slices.

Product-wide adoption needs a separate sequenced plan. Start with shared shell/navigation, then migrate Today, Programs, Vendors, Work, regulatory Imports, Explore and Configure according to task frequency and operational risk. Each slice must preserve domain and authority behavior, add its manifest entries and state fixtures, inspect full-host renders, repair the highest-impact defect and update this table. Until those rows are migrated, describe the result as “UI foundations, Sent forms and the template record surface migrated,” never “ClearSight is fully standardized.”
