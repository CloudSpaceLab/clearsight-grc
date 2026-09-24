BEGIN;

-- ropa_register_summary is a disposable projection and may be discarded during rollback.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM ropa_processing_activities)
    OR EXISTS (SELECT 1 FROM ropa_processing_activity_revisions)
    OR EXISTS (SELECT 1 FROM ropa_processing_activity_data_categories)
    OR EXISTS (SELECT 1 FROM ropa_processing_activity_recipients)
    OR EXISTS (SELECT 1 FROM ropa_processing_activity_systems)
    OR EXISTS (SELECT 1 FROM ropa_processing_activity_reviews)
    OR EXISTS (SELECT 1 FROM ropa_events)
  THEN
    RAISE EXCEPTION 'ROPA authoritative or history data exists; refusing to drop the register';
  END IF;
END $$;

DROP TABLE IF EXISTS ropa_register_summary;
DROP TRIGGER IF EXISTS ropa_event_immutable ON ropa_events;
DROP TRIGGER IF EXISTS ropa_event_aggregate_tenant_legal_entity_scope ON ropa_events;
DROP TRIGGER IF EXISTS ropa_event_actor_tenant_scope ON ropa_events;
DROP FUNCTION IF EXISTS protect_ropa_event();
DROP FUNCTION IF EXISTS validate_ropa_event_aggregate_scope();
DROP FUNCTION IF EXISTS validate_ropa_event_actor_scope();
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
