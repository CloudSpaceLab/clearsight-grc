BEGIN;

-- A governed form can support several records without changing its immutable origin.
-- MATTER is the first supported subject; additional types require their own scoped FK.
CREATE TABLE form_subject_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    subject_type text NOT NULL CHECK (subject_type = 'MATTER'),
    subject_id uuid NOT NULL,
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version > 0),
    linked_by uuid NOT NULL,
    linked_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    ended_at timestamptz,
    ended_by uuid,
    end_reason text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    FOREIGN KEY (subject_id, tenant_id, legal_entity_id) REFERENCES matters(id, tenant_id, legal_entity_id),
    FOREIGN KEY (tenant_id, legal_entity_id, form_template_id, form_template_version)
      REFERENCES monitoring_form_templates(tenant_id, legal_entity_id, id, version),
    FOREIGN KEY (linked_by, tenant_id) REFERENCES principals(id, tenant_id),
    FOREIGN KEY (ended_by, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT form_subject_links_end_ck CHECK (
      (ended_at IS NULL AND ended_by IS NULL AND end_reason IS NULL)
      OR (ended_at IS NOT NULL AND ended_by IS NOT NULL AND end_reason IS NOT NULL
          AND length(btrim(end_reason)) BETWEEN 1 AND 1000)
    )
);

CREATE UNIQUE INDEX form_subject_links_current_uq
  ON form_subject_links(tenant_id, legal_entity_id, subject_type, subject_id, form_template_id)
  WHERE ended_at IS NULL;

CREATE INDEX form_subject_links_subject_idx
  ON form_subject_links(tenant_id, legal_entity_id, subject_type, subject_id, linked_at DESC, id DESC)
  WHERE ended_at IS NULL;

CREATE TABLE form_subject_link_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    link_id uuid NOT NULL REFERENCES form_subject_links(id),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    event_type text NOT NULL CHECK (event_type IN ('LINKED', 'UNLINKED')),
    actor_id uuid NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload)='object'),
    FOREIGN KEY (actor_id, tenant_id) REFERENCES principals(id, tenant_id),
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id)
);
CREATE INDEX form_subject_link_events_link_idx
  ON form_subject_link_events(link_id, occurred_at, id);

COMMIT;
