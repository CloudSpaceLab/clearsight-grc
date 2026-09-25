BEGIN;
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM report_runs LIMIT 1) THEN
    RAISE EXCEPTION 'report runs exist; refusing to erase report history';
  END IF;
  IF EXISTS (SELECT 1 FROM report_definition_revisions LIMIT 1) THEN
    RAISE EXCEPTION 'report definition revisions exist; refusing to erase report history';
  END IF;
  IF EXISTS (SELECT 1 FROM report_definitions LIMIT 1) THEN
    RAISE EXCEPTION 'report definitions exist; refusing to erase report history';
  END IF;
END;
$$;
DROP INDEX IF EXISTS report_runs_queue_idx;
DROP TRIGGER IF EXISTS report_runs_generation_guard ON report_runs;
DROP FUNCTION IF EXISTS report_runs_generation_guard();
DROP TABLE IF EXISTS report_runs;
DROP TRIGGER IF EXISTS report_definition_revisions_immutable ON report_definition_revisions;
DROP FUNCTION IF EXISTS report_definition_revisions_immutable();
DROP TABLE IF EXISTS report_definition_revisions;
DROP TABLE IF EXISTS report_definitions;
DROP INDEX IF EXISTS ropa_matter_idx;
ALTER TABLE ropa_processing_activities DROP CONSTRAINT IF EXISTS ropa_activities_matter_tenant_fk;
ALTER TABLE ropa_processing_activities DROP COLUMN IF EXISTS matter_id;
COMMIT;
