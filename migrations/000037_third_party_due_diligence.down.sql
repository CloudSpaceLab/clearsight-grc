BEGIN;

DELETE FROM outbox_events WHERE aggregate_type='THIRD_PARTY_ASSESSMENT';
DELETE FROM third_party_events WHERE aggregate_type='THIRD_PARTY_ASSESSMENT';

DROP TRIGGER IF EXISTS third_party_event_aggregate_scope ON third_party_events;
DROP FUNCTION IF EXISTS third_party_event_aggregate_guard();
DROP INDEX IF EXISTS third_party_events_history_idx;
ALTER TABLE third_party_events
    DROP CONSTRAINT IF EXISTS third_party_events_typed_version_key,
    DROP CONSTRAINT IF EXISTS third_party_events_event_type_check,
    DROP CONSTRAINT IF EXISTS third_party_events_aggregate_type_check;
ALTER TABLE third_party_events
    ALTER COLUMN actor_principal_id SET NOT NULL,
    ADD CONSTRAINT third_party_events_aggregate_type_check CHECK (aggregate_type='VENDOR_RELATIONSHIP'),
    ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN ('VendorRelationshipCreated','VendorRelationshipUpdated')),
    ADD CONSTRAINT third_party_events_aggregate_id_tenant_id_fkey FOREIGN KEY (aggregate_id,tenant_id) REFERENCES third_party_relationships(id,tenant_id),
    ADD CONSTRAINT third_party_events_tenant_id_aggregate_id_aggregate_version_key UNIQUE (tenant_id,aggregate_id,aggregate_version);
CREATE INDEX third_party_events_history_idx
    ON third_party_events(tenant_id,aggregate_id,aggregate_version,id);

DROP TABLE IF EXISTS third_party_assessment_jobs;
DROP TABLE IF EXISTS third_party_documents;
DROP TABLE IF EXISTS third_party_assessment_reactions;
DROP TABLE IF EXISTS third_party_assessment_request_links;
DROP TABLE IF EXISTS third_party_assessment_matter_links;
DROP TABLE IF EXISTS third_party_assessments;

ALTER TABLE matters DROP CONSTRAINT matters_matter_type_check;
ALTER TABLE matters ADD CONSTRAINT matters_matter_type_check CHECK (matter_type IN (
    'REGULATORY_CHANGE','SUPERVISORY_FINDING','AUTHORITY_REQUEST','RISK_SITUATION','CONTROL_GAP','AUDIT_FINDING','EXCEPTION','INCIDENT',
    'OPERATIONAL_LOSS','DATA_BREACH','VENDOR_DEFICIENCY','CUSTOMER_CONCERN','OVERDUE_OBLIGATION','FAILED_VERIFICATION','EVIDENCE_CONTRADICTION','KRI_BREACH'
));
ALTER TABLE third_party_relationships DROP CONSTRAINT IF EXISTS third_party_relationships_scoped_id_key;
ALTER TABLE capture_artifacts DROP CONSTRAINT IF EXISTS capture_artifacts_id_tenant_request_key;
ALTER TABLE capture_submissions DROP CONSTRAINT IF EXISTS capture_submissions_id_tenant_request_key;
ALTER TABLE capture_invitations DROP CONSTRAINT IF EXISTS capture_invitations_id_tenant_request_key;

COMMIT;
