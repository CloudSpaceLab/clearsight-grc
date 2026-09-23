BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM ropa_processing_activity_revisions) THEN
    RAISE EXCEPTION 'ROPA processing activity history exists; refusing to drop the register';
  END IF;
END $$;

DROP TABLE IF EXISTS ropa_register_summary;
DROP TABLE IF EXISTS ropa_events;
DROP TABLE IF EXISTS ropa_processing_activity_reviews;
DROP TABLE IF EXISTS ropa_processing_activity_systems;
DROP TABLE IF EXISTS ropa_processing_activity_recipients;
DROP TABLE IF EXISTS ropa_processing_activity_data_categories;
DROP TRIGGER IF EXISTS ropa_revision_immutable ON ropa_processing_activity_revisions;
DROP FUNCTION IF EXISTS protect_ropa_revision();
DROP TRIGGER IF EXISTS ropa_legal_entity_immutable ON ropa_processing_activities;
DROP FUNCTION IF EXISTS protect_ropa_legal_entity();
DROP TABLE IF EXISTS ropa_processing_activity_revisions;
DROP TABLE IF EXISTS ropa_processing_activities;

COMMIT;
