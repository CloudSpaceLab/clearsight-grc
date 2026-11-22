# Vendor requested-document selection

The requested-document picker retains a selected document while loading another page. If a continuation is denied or the underlying record becomes unavailable, the selected document and previously loaded choices are now cleared and linking is disabled. Other temporary page failures retain the current selection and permit retry. Repeated cursor cycles stop pagination without appending the cyclic page; repeated occurrence IDs appear once.

Synthetic selected and denied states are retained in `docs/evidence/2026-11-22-vendor-collection-scope/` at 1440, 768 and 390 pixels. The evidence script checks the selected item is removed, linking is disabled and no horizontal overflow occurs. The empty-picker state from `docs/evidence/2026-11-20-v210-form-choices/` is the preceding visual baseline. The captures use simulated responses and do not establish server authorization. Legacy vendor checklist and onboarding test coverage remains in place.
