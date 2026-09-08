CREATE TABLE capture_distribution_creation_receipts (
 tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
 idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 240),
 payload_checksum text NOT NULL,
 distribution_id uuid NOT NULL REFERENCES capture_form_distributions(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(tenant_id,legal_entity_id,idempotency_key)
);
CREATE INDEX capture_vendor_form_requests_idx ON capture_requests(tenant_id,legal_entity_id,subject_id,updated_at DESC,id DESC) WHERE subject_type='VENDOR_RELATIONSHIP' AND form_template_id IS NOT NULL;
CREATE INDEX capture_vendor_workspace_progress_idx ON capture_response_workspace_edits(tenant_id,legal_entity_id,distribution_id,(patch->>'field_id'),result_version DESC,id DESC);
