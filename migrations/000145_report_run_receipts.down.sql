BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM report_run_receipts LIMIT 1) THEN
        RAISE EXCEPTION 'cannot remove report run receipts while receipts exist';
    END IF;
END $$;
DROP TABLE report_run_receipts;
DROP INDEX report_runs_receipt_identity_idx;
COMMIT;
