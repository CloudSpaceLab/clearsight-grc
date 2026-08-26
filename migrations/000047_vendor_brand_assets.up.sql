BEGIN;

ALTER TABLE third_parties
    ADD COLUMN website_domain text,
    ADD CONSTRAINT third_parties_website_domain_check CHECK (
        website_domain IS NULL OR (
            website_domain=btrim(website_domain)
            AND website_domain=lower(website_domain)
            AND char_length(website_domain) BETWEEN 1 AND 253
            AND website_domain ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$'
        )
    );

CREATE TABLE third_party_vendor_brand_assets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    vendor_id uuid NOT NULL,
    source_kind text NOT NULL CHECK (source_kind IN ('DISCOVERED','APPROVED_OVERRIDE')),
    state text NOT NULL CHECK (state IN ('CURRENT','SUPERSEDED')),
    source_domain text NOT NULL DEFAULT '' CHECK (
        source_domain=btrim(source_domain)
        AND char_length(source_domain) <= 253
        AND (source_domain='' OR source_domain ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$')
    ),
    artifact_key text NOT NULL CHECK (artifact_key=btrim(artifact_key) AND char_length(artifact_key) BETWEEN 1 AND 1024),
    source_digest text NOT NULL CHECK (source_digest ~ '^[0-9a-f]{64}$'),
    media_type text NOT NULL CHECK (media_type='image/png'),
    pixel_width integer NOT NULL CHECK (pixel_width BETWEEN 1 AND 4096),
    pixel_height integer NOT NULL CHECK (pixel_height BETWEEN 1 AND 4096),
    byte_size bigint NOT NULL CHECK (byte_size BETWEEN 1 AND 524288),
    retrieved_at timestamptz NOT NULL,
    next_refresh_at timestamptz,
    approved_by_principal_id uuid,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (id,tenant_id),
    FOREIGN KEY (vendor_id,tenant_id) REFERENCES third_parties(id,tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (approved_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((source_kind='DISCOVERED' AND source_domain<>'' AND approved_by_principal_id IS NULL)
        OR (source_kind='APPROVED_OVERRIDE' AND source_domain='' AND approved_by_principal_id IS NOT NULL)),
    CHECK (next_refresh_at IS NULL OR next_refresh_at > retrieved_at)
);
CREATE UNIQUE INDEX third_party_vendor_brand_assets_current_idx
    ON third_party_vendor_brand_assets(tenant_id,vendor_id,source_kind)
    WHERE state='CURRENT';
CREATE INDEX third_party_vendor_brand_assets_vendor_idx
    ON third_party_vendor_brand_assets(tenant_id,vendor_id,state,updated_at DESC,id DESC);

CREATE TABLE third_party_vendor_brand_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    vendor_id uuid NOT NULL,
    vendor_version bigint NOT NULL CHECK (vendor_version > 0),
    job_type text NOT NULL CHECK (job_type='DISCOVER_ICON'),
    website_domain text NOT NULL DEFAULT '' CHECK (
        website_domain=btrim(website_domain)
        AND char_length(website_domain) <= 253
        AND (website_domain='' OR website_domain ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$')
    ),
    state text NOT NULL CHECK (state IN ('READY','LEASED','COMPLETED','FAILED','CANCELLED')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 20),
    available_at timestamptz NOT NULL,
    lease_token uuid,
    lease_expires_at timestamptz,
    last_failure_code text NOT NULL DEFAULT '' CHECK (last_failure_code=btrim(last_failure_code) AND char_length(last_failure_code) <= 128),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (tenant_id,vendor_id),
    FOREIGN KEY (vendor_id,tenant_id) REFERENCES third_parties(id,tenant_id) ON DELETE CASCADE,
    CHECK ((state='LEASED' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR (state<>'LEASED' AND lease_token IS NULL AND lease_expires_at IS NULL)),
    CHECK ((state='CANCELLED' AND website_domain='') OR (state<>'CANCELLED' AND website_domain<>''))
);
CREATE INDEX third_party_vendor_brand_jobs_claim_idx
    ON third_party_vendor_brand_jobs(state,available_at,id)
    WHERE state IN ('READY','LEASED');

ALTER TABLE third_party_events
    DROP CONSTRAINT third_party_events_aggregate_type_check,
    DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events
    ADD CONSTRAINT third_party_events_aggregate_type_check CHECK (aggregate_type IN ('VENDOR','VENDOR_RELATIONSHIP','THIRD_PARTY_ASSESSMENT')),
    ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
        'VendorIdentityCreated','VendorIdentityUpdated','VendorRelationshipCreated','VendorRelationshipUpdated',
        'AssessmentStarted','AssessmentSetupCompleted','AssessmentSetupRetryQueued','AssessmentRequestPrepared','AssessmentRequestIssued',
        'AssessmentRequestReissuePrepared','AssessmentRequestReissued','AssessmentSubmitted','AssessmentReviewStarted',
        'AssessmentDeficiencyLinked','AssessmentDocumentValidated','AssessmentDocumentRejected','AssessmentCompleted','AssessmentCancelled'
    ));

CREATE OR REPLACE FUNCTION third_party_event_aggregate_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.aggregate_type='VENDOR' AND NOT EXISTS (
        SELECT 1 FROM third_parties WHERE id=NEW.aggregate_id AND tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'vendor identity event aggregate is outside tenant scope';
    END IF;
    IF NEW.aggregate_type='VENDOR_RELATIONSHIP' AND NOT EXISTS (
        SELECT 1 FROM third_party_relationships WHERE id=NEW.aggregate_id AND tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'third-party relationship event aggregate is outside tenant scope';
    END IF;
    IF NEW.aggregate_type='THIRD_PARTY_ASSESSMENT' AND NOT EXISTS (
        SELECT 1 FROM third_party_assessments WHERE id=NEW.aggregate_id AND tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'third-party assessment event aggregate is outside tenant scope';
    END IF;
    RETURN NEW;
END $$;

COMMIT;
