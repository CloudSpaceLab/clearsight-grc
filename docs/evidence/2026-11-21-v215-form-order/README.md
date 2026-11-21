# Form question ordering

The v2.15 form editor allows questions to move among siblings in the same section. A move is unavailable when it would place a conditional question or section before its source. Moving preserves the question identity, settings, scores and unsaved choice options; keyboard focus moves to the question's new position. The legacy section-order test now exercises this runtime rather than importing the removed builder. It remains one small part of the old builder's outstanding coverage.

The sample editor was rendered at 1440, 768 and 390 pixels with no horizontal overflow. The captured interaction moved a question, retained focus and checked a disabled boundary action. `scripts/v215-form-order-evidence.mjs` reproduces the sample render with the evidence preview; it does not establish backend authorization, dark-mode acceptance or complete form-builder parity. Earlier baseline renders are retained in `docs/evidence/2026-11-21-v215-form-authoring/`.
