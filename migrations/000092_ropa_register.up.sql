BEGIN;

CREATE TABLE ropa_processing_activities (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
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
  owner_principal_id uuid REFERENCES principals(id),
  required_authority_principal_id uuid REFERENCES principals(id),
  program_id uuid,
  version bigint NOT NULL CHECK(version>0),
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  UNIQUE (id, tenant_id),
  UNIQUE (id, tenant_id, legal_entity_id),
  UNIQUE (tenant_id, legal_entity_id, code),
  CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
CREATE UNIQUE INDEX ropa_activities_legal_entity_uk ON ropa_processing_activities(legal_entity_id,id);
CREATE INDEX ropa_register_keyset_idx ON ropa_processing_activities(tenant_id,legal_entity_id,status,next_review_date,id);
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
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
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
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_data_categories_activity_idx ON ropa_processing_activity_data_categories(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_recipients (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  recipient text NOT NULL,
  recipient_kind text NOT NULL DEFAULT 'EXTERNAL'
    CHECK (recipient_kind IN ('INTERNAL','EXTERNAL','AUTHORITY')),
  transfer_basis text NOT NULL DEFAULT '',
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,recipient),
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_recipients_activity_idx ON ropa_processing_activity_recipients(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_systems (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  system_name text NOT NULL,
  system_kind text NOT NULL DEFAULT 'APPLICATION'
    CHECK (system_kind IN ('APPLICATION','DATABASE','FILE','MANUAL','THIRD_PARTY')),
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,system_name),
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
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
  reviewer_principal_id uuid REFERENCES principals(id),
  created_at timestamptz NOT NULL,
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_reviews_due_idx ON ropa_processing_activity_reviews(tenant_id,legal_entity_id,due_date);

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
  FOREIGN KEY (tenant_id,legal_entity_id)
    REFERENCES legal_entities(tenant_id,id)
);
CREATE INDEX ropa_events_replay_idx ON ropa_events(tenant_id,aggregate_type,aggregate_id,aggregate_version);

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
  FOREIGN KEY (tenant_id,legal_entity_id) REFERENCES legal_entities(tenant_id,id)
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
