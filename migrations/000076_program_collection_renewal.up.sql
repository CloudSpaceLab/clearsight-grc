BEGIN;

ALTER TABLE monitoring_checks
    ADD COLUMN validity_months integer,
    ADD COLUMN renewal_window_days integer,
    ADD COLUMN reminder_count integer,
    ADD CONSTRAINT monitoring_checks_collection_policy_check CHECK (
        (validity_months IS NULL AND renewal_window_days IS NULL AND reminder_count IS NULL) OR
        (input_kind='FORM' AND validity_months BETWEEN 1 AND 120 AND renewal_window_days BETWEEN 1 AND 90
            AND renewal_window_days <= validity_months * 28 - 1 AND reminder_count BETWEEN 1 AND 5)
    );

ALTER TABLE capture_requests
    ADD COLUMN predecessor_request_id uuid,
    ADD COLUMN previous_responses jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT capture_requests_monitoring_lineage_check CHECK (
        origin_type IS DISTINCT FROM 'MONITORING_COLLECTION' OR (
            origin_version > 0 AND
            ((origin_version=1 AND predecessor_request_id IS NULL) OR
                (origin_version>1 AND predecessor_request_id IS NOT NULL))
        )
    ),
    ADD CONSTRAINT capture_requests_previous_responses_check CHECK (
        jsonb_typeof(previous_responses)='object' AND octet_length(previous_responses::text) <= 131072
    ),
    ADD CONSTRAINT capture_requests_predecessor_tenant_fk
        FOREIGN KEY(predecessor_request_id,tenant_id) REFERENCES capture_requests(id,tenant_id);

CREATE TABLE monitoring_collection_cycles (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    program_id uuid NOT NULL,
    monitoring_check_id uuid NOT NULL,
    monitoring_check_version bigint NOT NULL CHECK (monitoring_check_version > 0),
    sequence bigint NOT NULL CHECK (sequence > 0),
    validity_months integer NOT NULL CHECK (validity_months BETWEEN 1 AND 120),
    renewal_window_days integer NOT NULL CHECK (renewal_window_days BETWEEN 1 AND 90 AND renewal_window_days <= validity_months * 28 - 1),
    reminder_count integer NOT NULL CHECK (reminder_count BETWEEN 1 AND 5),
    reminders_sent integer NOT NULL DEFAULT 0 CHECK (reminders_sent BETWEEN 0 AND reminder_count),
    current_request_id uuid,
    predecessor_request_id uuid,
    latest_submission_id uuid,
    latest_submitted_at timestamptz,
    latest_respondent_label text NOT NULL DEFAULT ''
        CHECK (latest_respondent_label=btrim(latest_respondent_label) AND char_length(latest_respondent_label) <= 256),
    expires_at timestamptz NOT NULL,
    renewal_opens_at timestamptz NOT NULL,
    next_action_at timestamptz,
    recipient_route_type text NOT NULL CHECK (recipient_route_type IN ('INTERNAL_PRINCIPAL','EXTERNAL_CONTACT')),
    recipient_principal_id uuid,
    recipient_contact_ref text,
    recipient_safe_hint text NOT NULL DEFAULT '',
    delivery_state text NOT NULL DEFAULT 'NOT_DISPATCHED' CHECK (delivery_state IN ('NOT_DISPATCHED','ASSIGNED','DELIVERED','BLOCKED','FAILED')),
    delivery_reference text NOT NULL DEFAULT '' CHECK (delivery_reference=btrim(delivery_reference) AND char_length(delivery_reference) <= 512),
    state text NOT NULL CHECK (state IN ('SCHEDULED','CLAIMED','AWAITING_RESPONSE','COMPLETE','CANCELLED','BLOCKED','FAILED')),
    lease_owner text,
    lease_token uuid,
    lease_until timestamptz,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 20),
    safe_error text NOT NULL DEFAULT '' CHECK (safe_error=btrim(safe_error) AND char_length(safe_error) <= 1000),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE(tenant_id,monitoring_check_id,sequence),
    UNIQUE(id,tenant_id),
    FOREIGN KEY(program_id,tenant_id) REFERENCES programs(id,tenant_id),
    FOREIGN KEY(tenant_id,monitoring_check_id,monitoring_check_version) REFERENCES monitoring_checks(tenant_id,id,version),
    FOREIGN KEY(current_request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY(predecessor_request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY(latest_submission_id,tenant_id,current_request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
    FOREIGN KEY(recipient_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((latest_submission_id IS NULL AND latest_submitted_at IS NULL) OR
        (latest_submission_id IS NOT NULL AND latest_submitted_at IS NOT NULL AND current_request_id IS NOT NULL)),
    CHECK ((recipient_route_type='INTERNAL_PRINCIPAL' AND recipient_principal_id IS NOT NULL AND recipient_contact_ref IS NULL AND recipient_safe_hint='') OR
        (recipient_route_type='EXTERNAL_CONTACT' AND recipient_principal_id IS NULL AND recipient_contact_ref IS NOT NULL
            AND recipient_contact_ref=btrim(recipient_contact_ref) AND char_length(recipient_contact_ref) BETWEEN 1 AND 512
            AND recipient_safe_hint=btrim(recipient_safe_hint) AND char_length(recipient_safe_hint) BETWEEN 1 AND 256)),
    CHECK ((state='CLAIMED' AND lease_owner IS NOT NULL AND btrim(lease_owner)<>'' AND lease_token IS NOT NULL AND lease_until IS NOT NULL) OR
        (state<>'CLAIMED' AND lease_owner IS NULL AND lease_token IS NULL AND lease_until IS NULL)),
    CHECK (renewal_opens_at < expires_at),
    CHECK (next_action_at IS NULL OR next_action_at <= expires_at)
);

CREATE INDEX monitoring_collection_cycles_due_idx
    ON monitoring_collection_cycles(next_action_at,id)
    WHERE state IN ('SCHEDULED','CLAIMED','AWAITING_RESPONSE') AND next_action_at IS NOT NULL;
CREATE INDEX monitoring_collection_cycles_program_idx
    ON monitoring_collection_cycles(tenant_id,program_id,monitoring_check_id,sequence DESC,id);

COMMIT;
