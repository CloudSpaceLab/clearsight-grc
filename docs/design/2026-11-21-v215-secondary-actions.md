# Secondary action tokens

Search, table and panel secondary actions use the shared color, height and radius tokens. Existing light colors and the 44-pixel minimum height are preserved. Other action styles and dark-theme mappings are not yet migrated.

The existing field-token browser check captures sample Programs and vendor intake at 1440, 768 and 390 pixels. It verifies the search action's computed colors and geometry, and confirms that a semantic surface override reaches the action and can be removed. Local captures are in `docs/evidence/2026-11-21-v215-secondary-actions/`; prior captures are in `docs/evidence/2026-11-21-v215-field-tokens/`. Inspection of the 390-pixel Programs render found the search action and content readable without horizontal overflow. This is component evidence, not full-screen or production acceptance.
