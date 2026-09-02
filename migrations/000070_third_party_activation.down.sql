BEGIN;

-- Activation is a material, append-only fact. A rollback must not delete its
-- policy, receipt or event history, nor reintroduce a constraint that rejects
-- already-recorded activation events. Runtime rollback is performed by a new
-- independently approved policy revision; this migration intentionally keeps
-- the reconstructable records.

COMMIT;
