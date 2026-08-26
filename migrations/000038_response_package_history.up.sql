BEGIN;

CREATE INDEX continuity_response_package_history_idx
ON continuity_events (tenant_id, aggregate_id, ((payload->>'id')), aggregate_version DESC)
WHERE aggregate_type='MATTER'
  AND event_type IN ('RESPONSE_PACKAGE_ADDED','RESPONSE_PACKAGE_STATE_CHANGED');

COMMIT;
