BEGIN;

ALTER TABLE third_party_vendor_brand_assets ADD COLUMN asset_token text;
UPDATE third_party_vendor_brand_assets SET asset_token=replace(uuidv7()::text,'-','')||replace(uuidv7()::text,'-','');
ALTER TABLE third_party_vendor_brand_assets ALTER COLUMN asset_token SET NOT NULL;
ALTER TABLE third_party_vendor_brand_assets ADD CONSTRAINT third_party_vendor_brand_assets_token_check CHECK(asset_token ~ '^[0-9a-f]{64}$');
CREATE UNIQUE INDEX third_party_vendor_brand_assets_token_idx ON third_party_vendor_brand_assets(tenant_id,vendor_id,asset_token);

CREATE TABLE third_party_vendor_brand_upload_reservations (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    vendor_id uuid NOT NULL,
    idempotency_key text NOT NULL CHECK (idempotency_key=btrim(idempotency_key) AND char_length(idempotency_key) BETWEEN 1 AND 200),
    expected_brand_version bigint NOT NULL CHECK (expected_brand_version >= 0),
    artifact_key text NOT NULL CHECK (artifact_key=btrim(artifact_key) AND char_length(artifact_key) BETWEEN 1 AND 1024),
    source_digest text NOT NULL CHECK (source_digest ~ '^[0-9a-f]{64}$'),
    state text NOT NULL CHECK (state IN ('RESERVED','CLEANING','COMMITTED')),
    lease_token uuid,
    lease_expires_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    PRIMARY KEY (tenant_id,vendor_id,idempotency_key),
    FOREIGN KEY (vendor_id,tenant_id) REFERENCES third_parties(id,tenant_id) ON DELETE CASCADE,
    CHECK ((state='CLEANING' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL) OR (state<>'CLEANING' AND lease_token IS NULL AND lease_expires_at IS NULL))
);
CREATE INDEX third_party_vendor_brand_upload_cleanup_idx
    ON third_party_vendor_brand_upload_reservations(updated_at,tenant_id,vendor_id)
    WHERE state='RESERVED';
CREATE INDEX third_party_vendor_brand_upload_artifact_idx ON third_party_vendor_brand_upload_reservations(tenant_id,artifact_key);

CREATE TABLE third_party_vendor_brand_command_receipts (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    vendor_id uuid NOT NULL,
    idempotency_key text NOT NULL CHECK (idempotency_key=btrim(idempotency_key) AND char_length(idempotency_key) BETWEEN 1 AND 200),
    command_type text NOT NULL CHECK (command_type IN ('thirdparty.vendor.brand.approve','thirdparty.vendor.brand.remove')),
    expected_brand_version bigint NOT NULL CHECK (expected_brand_version >= 0),
    result_brand_version bigint NOT NULL CHECK (result_brand_version > expected_brand_version),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id,vendor_id,idempotency_key),
    FOREIGN KEY (vendor_id,tenant_id) REFERENCES third_parties(id,tenant_id) ON DELETE CASCADE
);

ALTER TABLE third_party_events DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
    'VendorIdentityCreated','VendorIdentityUpdated','VendorBrandDiscovered','VendorBrandApproved','VendorBrandRemoved',
    'VendorRelationshipCreated','VendorRelationshipUpdated','AssessmentStarted','AssessmentSetupCompleted',
    'AssessmentSetupRetryQueued','AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared',
    'AssessmentRequestReissued','AssessmentSubmitted','AssessmentReviewStarted','AssessmentDeficiencyLinked',
    'AssessmentDocumentValidated','AssessmentDocumentRejected','AssessmentCompleted','AssessmentCancelled'
));

COMMIT;
