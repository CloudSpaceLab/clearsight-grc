# Documents: selection focus

Opening a submitted document moves keyboard focus to its detail heading so the selected document and preview are reachable after the inventory, including when the panel stacks below the list on narrow screens. Closing the panel returns focus to the exact title or Details button that opened it, if that button remains present. The selected occurrence and protected preview retain the existing scoped inventory and content contracts; no new read, write, or authorization behavior is introduced.

The design reference is `clearsight-v2.15/documents.html` (SHA-256 `31e87e1c9e97c9daa4118e6c2cd257e29636a5a76ade0a3fefeb58da646d5508`). The earlier baseline and sample selected-document renders are in `docs/evidence/2026-11-21-v215-documents/`. This change does not provide standalone uploads, folders, file approval, or restore selection across routes.

The synthetic fixture was checked at 1440, 768 and 390 pixels using Chrome keyboard activation and the close button. The detail heading received focus and remained below the fixed header, the narrow layout scrolled the panel into view, closing restored the initiating button, and no horizontal overflow was observed. Captures and geometry are in `docs/evidence/2026-11-21-v215-document-focus/`; the image bytes are a synthetic 1×1 pixel sample, so they do not establish real file preview quality, live authorization, dark-theme or whole-screen acceptance.

The follow-up render exercises reopening the same selection and moving from details to the form-filtered inventory. Keyboard focus reached the detail heading and the active form filter respectively at 1440, 768 and 390 pixels, with no horizontal overflow; images and results are in `docs/evidence/2026-11-21-v215-document-focus/`. The fixture is isolated sample data.

The Documents and response lists serialize each load-more request per query. A changed query starts a fresh generation, and a late page from the previous query is ignored. The response review form disables its decision and comment during save; earlier-history loading cannot overlap a review command, and reloading a response clears pending document/history loading states.
