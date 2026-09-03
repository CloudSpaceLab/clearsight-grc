BEGIN;

DO $$
BEGIN
    IF to_regclass('public.audit_export_receipts') IS NOT NULL
       AND EXISTS (SELECT 1 FROM audit_export_receipts LIMIT 1) THEN
        RAISE EXCEPTION 'audit_export_receipts is not empty; preserve export audit history before downgrade';
    END IF;
END $$;

DROP TABLE IF EXISTS audit_export_receipts;

COMMIT;
