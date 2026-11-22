# Vendor assessment review start

Starting review now confirms that the returned assessment matches the current vendor, tenant, legal entity, version, request and submission, and includes a reviewer and review-start timestamp. Unconfirmed outcomes require an assessment reload; access denial clears assessment details before a fresh scoped read. A scope change discards a pending receipt and releases the new vendor's action state.

Synthetic prior, unconfirmed and access-denied states are retained in `docs/evidence/2026-11-22-vendor-start-review/` at 1440, 768 and 390 pixels. The evidence script checks the primary action is withheld after an unconfirmed result, the previous answer is hidden after access denial and no horizontal overflow occurs. The images are not evidence of production authorization. Existing legacy vendor onboarding tests remain until their remaining material behaviors are migrated.
The vendor detail is keyed by tenant, legal entity, relationship and assessment, so an old assessment is not rendered during a scope change. A layout-phase regression covers those changes; the application shell also keys the vendor workspace by actor.
