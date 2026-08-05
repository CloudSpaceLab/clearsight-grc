BEGIN;

DROP INDEX IF EXISTS matters_search_idx;
DROP INDEX IF EXISTS matters_open_summary_order_idx;
DROP INDEX IF EXISTS matters_summary_order_idx;
DROP INDEX IF EXISTS programs_search_idx;
DROP INDEX IF EXISTS programs_summary_order_idx;

ALTER TABLE matters DROP COLUMN IF EXISTS search_document;
ALTER TABLE programs DROP COLUMN IF EXISTS search_document;

COMMIT;
