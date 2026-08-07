BEGIN;

-- These foundation-era tables have no current runtime owner. Fail closed if a
-- deployment has written data into either table so unsupported use cannot be
-- destroyed silently during cleanup.
DO $$
BEGIN
    IF to_regclass('public.audit_events') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM public.audit_events LIMIT 1) THEN
            RAISE EXCEPTION 'audit_events is not empty; classify and migrate its data before removal';
        END IF;
    END IF;

    IF to_regclass('public.readiness_snapshots') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM public.readiness_snapshots LIMIT 1) THEN
            RAISE EXCEPTION 'readiness_snapshots is not empty; classify and migrate its data before removal';
        END IF;
    END IF;
END
$$;

DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS readiness_snapshots;

COMMIT;
