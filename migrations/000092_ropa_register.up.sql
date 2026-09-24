BEGIN;

CREATE TABLE ropa_processing_activities (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  legal_entity_id uuid NOT NULL,
  code text NOT NULL,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('NEW','OPEN','CLOSED')),
  purpose text NOT NULL DEFAULT '',
  lawful_basis text NOT NULL DEFAULT '',
  controller text NOT NULL,
  processor text NOT NULL DEFAULT '',
  automated_decision_making boolean NOT NULL DEFAULT false,
  data_subject_categories text NOT NULL DEFAULT '',
  personal_data_categories text NOT NULL DEFAULT '',
  security_measures text NOT NULL DEFAULT '',
  retention_period text NOT NULL DEFAULT '',
  start_date date,
  end_date date,
  next_review_date date,
  owner_principal_id uuid,
  required_authority_principal_id uuid,
  program_id uuid,
  version bigint NOT NULL CHECK(version>0),
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  UNIQUE (id, tenant_id, legal_entity_id),
  UNIQUE (tenant_id, legal_entity_id, code),
  CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date),
  CONSTRAINT ropa_processing_activities_updated_at_order_ck
    CHECK (updated_at >= created_at),
  CONSTRAINT ropa_processing_activities_legal_entity_tenant_fk
    FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
  CONSTRAINT ropa_processing_activities_owner_tenant_fk
    FOREIGN KEY (owner_principal_id, tenant_id) REFERENCES principals(id, tenant_id),
  CONSTRAINT ropa_processing_activities_required_authority_tenant_fk
    FOREIGN KEY (required_authority_principal_id, tenant_id) REFERENCES principals(id, tenant_id),
  -- Migration 000040 adds programs_id_tenant_entity_key; use that exact
  -- three-column key so a continuity link cannot cross legal entities.
  CONSTRAINT ropa_processing_activities_program_scope_fk
    FOREIGN KEY (program_id, tenant_id, legal_entity_id) REFERENCES programs(id, tenant_id, legal_entity_id)
);
-- The repository ORDER BY and cursor predicates must use this exact expression for both indexes:
-- (CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
--  COALESCE(next_review_date, '0001-01-01'::date), id).
CREATE INDEX ropa_register_keyset_idx ON ropa_processing_activities(
  tenant_id,
  legal_entity_id,
  (CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END),
  (COALESCE(next_review_date, '0001-01-01'::date)),
  id
) WHERE end_date IS NULL;
CREATE INDEX ropa_register_history_keyset_idx ON ropa_processing_activities(
  tenant_id,
  legal_entity_id,
  (CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END),
  (COALESCE(next_review_date, '0001-01-01'::date)),
  id
);
CREATE INDEX ropa_lawful_basis_idx ON ropa_processing_activities(tenant_id,legal_entity_id,lawful_basis);
CREATE INDEX ropa_owner_idx ON ropa_processing_activities(tenant_id,legal_entity_id,owner_principal_id);

CREATE TABLE ropa_processing_activity_revisions (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  version bigint NOT NULL CHECK(version>0),
  snapshot jsonb NOT NULL,
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,version),
  CONSTRAINT ropa_processing_activity_revisions_snapshot_object_ck
    CHECK (jsonb_typeof(snapshot) = 'object'),
  FOREIGN KEY (activity_id, tenant_id, legal_entity_id)
    REFERENCES ropa_processing_activities(id, tenant_id, legal_entity_id)
);
CREATE FUNCTION protect_ropa_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'ROPA processing activity history is immutable'; END; $$;
CREATE TRIGGER ropa_revision_immutable BEFORE UPDATE OR DELETE ON ropa_processing_activity_revisions FOR EACH ROW EXECUTE FUNCTION protect_ropa_revision();

CREATE TABLE ropa_processing_activity_data_categories (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  category text NOT NULL,
  sensitivity text NOT NULL DEFAULT 'UNCLASSIFIED'
    CHECK (sensitivity IN ('UNCLASSIFIED','DIRECT_PERSONAL','INDIRECT_PERSONAL','SENSITIVE_BY_NATURE','SENSITIVE_BY_LAW')),
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,category),
  FOREIGN KEY (activity_id, tenant_id, legal_entity_id)
    REFERENCES ropa_processing_activities(id, tenant_id, legal_entity_id)
);
CREATE INDEX ropa_data_categories_activity_idx ON ropa_processing_activity_data_categories(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_recipients (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  recipient text NOT NULL,
  recipient_kind text NOT NULL DEFAULT 'EXTERNAL'
    CHECK (recipient_kind IN ('INTERNAL','EXTERNAL','AUTHORITY')),
  country_code text,
  is_cross_border boolean NOT NULL DEFAULT false,
  -- This vocabulary maps to the NDPA Article 45 and Schedule 5 safeguards for cross-border transfers.
  transfer_basis text NOT NULL DEFAULT 'NOT_APPLICABLE',
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,recipient),
  CONSTRAINT ropa_recipients_country_code_ck
    CHECK (country_code IS NULL OR country_code ~ '^[A-Z]{2}$'),
  CONSTRAINT ropa_recipients_transfer_basis_ck
    CHECK (transfer_basis IN ('ADEQUACY','APPROVED_INSTRUMENT','RECOGNISED_LAWFUL_BASIS','CONSENT','STANDARD_CONTRACT_CLAUSES','BINDING_CORPORATE_RULES','CERTIFICATION','NOT_APPLICABLE')),
  CONSTRAINT ropa_recipients_cross_border_coherence_ck
    CHECK (
      (is_cross_border
        AND country_code IS NOT NULL
        AND transfer_basis <> 'NOT_APPLICABLE')
      OR
      (NOT is_cross_border
        AND country_code IS NULL
        AND transfer_basis = 'NOT_APPLICABLE')
    ),
  FOREIGN KEY (activity_id, tenant_id, legal_entity_id)
    REFERENCES ropa_processing_activities(id, tenant_id, legal_entity_id)
);
CREATE INDEX ropa_recipients_activity_idx ON ropa_processing_activity_recipients(tenant_id,legal_entity_id,activity_id);
CREATE INDEX ropa_recipients_cross_border_idx ON ropa_processing_activity_recipients(tenant_id,legal_entity_id,is_cross_border);

CREATE TABLE ropa_processing_activity_systems (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  system_name text NOT NULL,
  system_kind text NOT NULL DEFAULT 'APPLICATION'
    CHECK (system_kind IN ('APPLICATION','DATABASE','FILE','MANUAL','THIRD_PARTY')),
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,system_name),
  FOREIGN KEY (activity_id, tenant_id, legal_entity_id)
    REFERENCES ropa_processing_activities(id, tenant_id, legal_entity_id)
);
CREATE INDEX ropa_systems_activity_idx ON ropa_processing_activity_systems(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_reviews (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  due_date date NOT NULL,
  completed_at timestamptz,
  outcome text CHECK (outcome IS NULL OR outcome IN ('CONFIRMED','REVISED','WITHDRAWN')),
  reviewer_principal_id uuid,
  created_at timestamptz NOT NULL,
  CONSTRAINT ropa_processing_activity_reviews_reviewer_tenant_fk
    FOREIGN KEY (reviewer_principal_id, tenant_id) REFERENCES principals(id, tenant_id),
  FOREIGN KEY (activity_id, tenant_id, legal_entity_id)
    REFERENCES ropa_processing_activities(id, tenant_id, legal_entity_id)
);
CREATE INDEX ropa_reviews_due_idx ON ropa_processing_activity_reviews(tenant_id,legal_entity_id,due_date);
CREATE INDEX ropa_reviews_activity_idx ON ropa_processing_activity_reviews(tenant_id,legal_entity_id,activity_id,created_at,id);

CREATE TABLE ropa_events (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  legal_entity_id uuid NOT NULL,
  aggregate_type text NOT NULL CHECK (aggregate_type IN ('PROCESSING_ACTIVITY','REPORT_DEFINITION','REPORT_RUN')),
  aggregate_id uuid NOT NULL,
  aggregate_version bigint NOT NULL CHECK(aggregate_version>0),
  type text NOT NULL,
  payload jsonb NOT NULL,
  actor_type text NOT NULL CHECK (actor_type IN ('USER','SERVICE')),
  actor_id uuid,
  occurred_at timestamptz NOT NULL,
  UNIQUE (tenant_id,aggregate_type,aggregate_id,aggregate_version),
  CONSTRAINT ropa_events_payload_object_ck
    CHECK (jsonb_typeof(payload) = 'object'),
  FOREIGN KEY (legal_entity_id, tenant_id)
    REFERENCES legal_entities(id, tenant_id)
);

CREATE FUNCTION validate_ropa_event_actor_scope() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.actor_type = 'SERVICE' THEN
    IF NEW.actor_id IS NOT NULL THEN
      RAISE EXCEPTION 'ROPA SERVICE event actor_id must be NULL' USING ERRCODE = '23514';
    END IF;
  ELSIF NEW.actor_type = 'USER' THEN
    IF NEW.actor_id IS NULL OR NOT EXISTS (
      SELECT 1
      FROM principals p
      WHERE p.id = NEW.actor_id
        AND p.tenant_id = NEW.tenant_id
    ) THEN
      RAISE EXCEPTION 'ROPA USER event actor is outside the event tenant' USING ERRCODE = '23514';
    END IF;
  ELSE
    RAISE EXCEPTION 'ROPA event actor_type is invalid' USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END; $$;
CREATE TRIGGER ropa_event_actor_tenant_scope BEFORE INSERT ON ropa_events
  FOR EACH ROW EXECUTE FUNCTION validate_ropa_event_actor_scope();

-- The aggregate-scope trigger also closes the event-payload identity/version
-- invariant before an event can enter the append-only ledger.
CREATE FUNCTION validate_ropa_event_aggregate_scope() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.aggregate_type = 'PROCESSING_ACTIVITY' THEN
    IF NOT EXISTS (
      SELECT 1
      FROM ropa_processing_activities a
      WHERE a.id = NEW.aggregate_id
        AND a.tenant_id = NEW.tenant_id
        AND a.legal_entity_id = NEW.legal_entity_id
        AND a.version = NEW.aggregate_version
    ) THEN
      RAISE EXCEPTION 'ROPA processing activity event aggregate is outside the event scope' USING ERRCODE = '23514';
    END IF;
    IF NEW.payload->>'id' IS DISTINCT FROM NEW.aggregate_id::text
       OR NEW.payload->>'tenant_id' IS DISTINCT FROM NEW.tenant_id::text
       OR NEW.payload->>'legal_entity_id' IS DISTINCT FROM NEW.legal_entity_id::text
       OR NEW.payload->>'version' IS DISTINCT FROM NEW.aggregate_version::text THEN
      RAISE EXCEPTION 'ROPA processing activity event payload scope/version does not match the event row' USING ERRCODE = '23514';
    END IF;
    IF NEW.type = 'processing_activity.created' AND NEW.aggregate_version <> 1 THEN
      RAISE EXCEPTION 'ROPA processing activity create event must start at version 1' USING ERRCODE = '23514';
    END IF;
    IF NEW.type IN ('processing_activity.updated','processing_activity.transitioned')
       AND NEW.aggregate_version <= 1 THEN
      RAISE EXCEPTION 'ROPA processing activity non-create event cannot start at version 1' USING ERRCODE = '23514';
    END IF;
    IF NEW.type NOT IN ('processing_activity.created','processing_activity.updated','processing_activity.transitioned') THEN
      RAISE EXCEPTION 'ROPA processing activity event type is invalid' USING ERRCODE = '23514';
    END IF;
  END IF;
  RETURN NEW;
END; $$;
CREATE TRIGGER ropa_event_aggregate_tenant_legal_entity_scope BEFORE INSERT ON ropa_events
  FOR EACH ROW EXECUTE FUNCTION validate_ropa_event_aggregate_scope();

CREATE FUNCTION protect_ropa_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'ROPA event history is immutable';
  RETURN NEW;
END; $$;
CREATE TRIGGER ropa_event_immutable BEFORE UPDATE OR DELETE ON ropa_events
  FOR EACH ROW EXECUTE FUNCTION protect_ropa_event();

CREATE TABLE ropa_register_summary (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  generated_at timestamptz NOT NULL,
  projection_version text NOT NULL,
  source_high_water timestamptz NOT NULL,
  population integer NOT NULL,
  excluded integer,
  unknown integer,
  counts jsonb NOT NULL,
  PRIMARY KEY(tenant_id,legal_entity_id),
  CONSTRAINT ropa_register_summary_population_nonnegative_ck
    CHECK (population >= 0),
  CONSTRAINT ropa_register_summary_excluded_nonnegative_ck
    CHECK (excluded IS NULL OR excluded >= 0),
  CONSTRAINT ropa_register_summary_unknown_nonnegative_ck
    CHECK (unknown IS NULL OR unknown >= 0),
  CONSTRAINT ropa_register_summary_counts_object_ck
    CHECK (jsonb_typeof(counts) = 'object'),
  CONSTRAINT ropa_register_summary_projection_version_nonblank_ck
    CHECK (btrim(projection_version) <> ''),
  FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id)
);

CREATE FUNCTION protect_ropa_legal_entity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.legal_entity_id IS DISTINCT FROM OLD.legal_entity_id THEN
    RAISE EXCEPTION 'ROPA processing activity legal entity is immutable';
  END IF;
  RETURN NEW;
END; $$;
CREATE TRIGGER ropa_legal_entity_immutable BEFORE UPDATE ON ropa_processing_activities
  FOR EACH ROW EXECUTE FUNCTION protect_ropa_legal_entity();

COMMIT;
