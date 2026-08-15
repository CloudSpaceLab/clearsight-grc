BEGIN;

ALTER TABLE source_observations
    ADD COLUMN scope_kind text NOT NULL DEFAULT 'SOURCE',
    ADD COLUMN connection_id uuid,
    ADD COLUMN connection_version bigint,
    ADD COLUMN view_id uuid,
    ADD COLUMN view_version bigint,
    ADD COLUMN binding_id uuid,
    ADD COLUMN binding_version bigint;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM source_observations so
          JOIN evidence_sources es ON es.id=so.source_id
         WHERE es.tenant_id<>so.tenant_id
    ) THEN
        RAISE EXCEPTION 'legacy source observations contain cross-tenant source provenance; repair before applying source-access runtime state';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM source_observations so
          JOIN principals p ON p.id=so.recorded_by
         WHERE so.recorded_by IS NOT NULL
           AND p.tenant_id<>so.tenant_id
    ) THEN
        RAISE EXCEPTION 'legacy source observations contain cross-tenant recorder provenance; repair before applying source-access runtime state';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM source_observations so
         WHERE so.observed_at > clock_timestamp() + interval '5 minutes'
    ) THEN
        RAISE EXCEPTION 'legacy source observations exceed the permitted clock skew; repair before applying source-access runtime state';
    END IF;
END
$$;

ALTER TABLE source_observations
    ADD CONSTRAINT source_observations_scope_kind_ck
        CHECK (scope_kind IN ('SOURCE','CONNECTION','VIEW','BINDING')),
    ADD CONSTRAINT source_observations_scope_shape_ck
        CHECK (
            (scope_kind='SOURCE'
                AND connection_id IS NULL AND connection_version IS NULL
                AND view_id IS NULL AND view_version IS NULL
                AND binding_id IS NULL AND binding_version IS NULL)
            OR
            (scope_kind='CONNECTION'
                AND connection_id IS NOT NULL AND connection_version > 0
                AND view_id IS NULL AND view_version IS NULL
                AND binding_id IS NULL AND binding_version IS NULL)
            OR
            (scope_kind='VIEW'
                AND connection_id IS NOT NULL AND connection_version > 0
                AND view_id IS NOT NULL AND view_version > 0
                AND binding_id IS NULL AND binding_version IS NULL)
            OR
            (scope_kind='BINDING'
                AND connection_id IS NOT NULL AND connection_version > 0
                AND view_id IS NOT NULL AND view_version > 0
                AND binding_id IS NOT NULL AND binding_version > 0)
        ),
    ADD CONSTRAINT source_observations_connection_fk
        FOREIGN KEY (connection_id,tenant_id,source_id,connection_version)
        REFERENCES source_connections(connection_id,tenant_id,source_id,version),
    ADD CONSTRAINT source_observations_view_fk
        FOREIGN KEY (view_id,tenant_id,source_id,view_version)
        REFERENCES source_views(view_id,tenant_id,source_id,version),
    ADD CONSTRAINT source_observations_binding_fk
        FOREIGN KEY (binding_id,tenant_id,source_id,binding_version)
        REFERENCES source_bindings(binding_id,tenant_id,source_id,version);

CREATE FUNCTION source_observation_scope_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    expected_connection_id uuid;
    expected_connection_version bigint;
    expected_view_id uuid;
    expected_view_version bigint;
BEGIN
    IF NEW.observed_at > clock_timestamp() + interval '5 minutes' THEN
        RAISE EXCEPTION 'source observation time exceeds the permitted clock skew' USING ERRCODE='23514';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM evidence_sources es
         WHERE es.id=NEW.source_id
           AND es.tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'source observation source does not belong to the supplied tenant' USING ERRCODE='23514';
    END IF;
    IF NEW.recorded_by IS NOT NULL AND NOT EXISTS (
        SELECT 1
          FROM principals p
         WHERE p.id=NEW.recorded_by
           AND p.tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'source observation recorder does not belong to the supplied tenant' USING ERRCODE='23514';
    END IF;
    IF NEW.scope_kind IN ('VIEW','BINDING') THEN
        SELECT sv.connection_id,sv.connection_version
          INTO expected_connection_id,expected_connection_version
          FROM source_views sv
         WHERE sv.tenant_id=NEW.tenant_id
           AND sv.source_id=NEW.source_id
           AND sv.view_id=NEW.view_id
           AND sv.version=NEW.view_version;
        IF expected_connection_id IS NULL
           OR expected_connection_id<>NEW.connection_id
           OR expected_connection_version<>NEW.connection_version THEN
            RAISE EXCEPTION 'source observation view does not belong to the supplied connection revision' USING ERRCODE='23514';
        END IF;
    END IF;
    IF NEW.scope_kind='BINDING' THEN
        SELECT sb.view_id,sb.view_version
          INTO expected_view_id,expected_view_version
          FROM source_bindings sb
         WHERE sb.tenant_id=NEW.tenant_id
           AND sb.source_id=NEW.source_id
           AND sb.binding_id=NEW.binding_id
           AND sb.version=NEW.binding_version;
        IF expected_view_id IS NULL
           OR expected_view_id<>NEW.view_id
           OR expected_view_version<>NEW.view_version THEN
            RAISE EXCEPTION 'source observation binding does not belong to the supplied view revision' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER source_observation_scope_guard_trigger
BEFORE INSERT OR UPDATE ON source_observations
FOR EACH ROW EXECUTE FUNCTION source_observation_scope_guard();

CREATE INDEX source_observations_connection_history_idx
    ON source_observations(tenant_id,source_id,connection_id,connection_version,observed_at DESC,id DESC)
    WHERE scope_kind='CONNECTION';
CREATE INDEX source_observations_view_history_idx
    ON source_observations(tenant_id,source_id,view_id,view_version,observed_at DESC,id DESC)
    WHERE scope_kind='VIEW';
CREATE INDEX source_observations_binding_history_idx
    ON source_observations(tenant_id,source_id,binding_id,binding_version,observed_at DESC,id DESC)
    WHERE scope_kind='BINDING';

CREATE TABLE source_binding_checkpoints (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id uuid NOT NULL,
    binding_id uuid NOT NULL,
    binding_version bigint NOT NULL CHECK (binding_version > 0),
    position_kind text NOT NULL DEFAULT '' CHECK (position_kind IN ('','CURSOR','ETAG','WATERMARK','EVENT_ID')),
    position_value text NOT NULL DEFAULT '' CHECK (octet_length(position_value) <= 16384 AND NOT (position_value ~ '[[:cntrl:]]')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    locked_by text NOT NULL DEFAULT '' CHECK (locked_by=btrim(locked_by) AND octet_length(locked_by) <= 128 AND NOT (locked_by ~ '[[:cntrl:]]')),
    lease_until timestamptz,
    next_attempt_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    last_error_code text NOT NULL DEFAULT '' CHECK (last_error_code=btrim(last_error_code) AND octet_length(last_error_code) <= 128 AND NOT (last_error_code ~ '[[:cntrl:]]')),
    failed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    PRIMARY KEY (tenant_id,binding_id,binding_version),
    FOREIGN KEY (binding_id,tenant_id,source_id,binding_version)
        REFERENCES source_bindings(binding_id,tenant_id,source_id,version) ON DELETE CASCADE,
    CHECK (
        (position_kind='' AND position_value='')
        OR (position_kind<>'' AND position_value<>'' AND position_value=btrim(position_value))
    ),
    CHECK (
        (locked_by='' AND lease_until IS NULL)
        OR (locked_by<>'' AND lease_until IS NOT NULL)
    ),
    CHECK (failed_at IS NULL OR (locked_by='' AND lease_until IS NULL))
);
CREATE INDEX source_binding_checkpoints_due_idx
    ON source_binding_checkpoints(next_attempt_at,binding_id,binding_version)
    WHERE failed_at IS NULL;
CREATE INDEX source_binding_checkpoints_source_idx
    ON source_binding_checkpoints(tenant_id,source_id,binding_id,binding_version);

CREATE FUNCTION source_binding_checkpoint_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    binding_current boolean;
    binding_status text;
    binding_operations jsonb;
BEGIN
    IF TG_OP='UPDATE' AND ROW(OLD.tenant_id,OLD.source_id,OLD.binding_id,OLD.binding_version,OLD.created_at)
        IS DISTINCT FROM ROW(NEW.tenant_id,NEW.source_id,NEW.binding_id,NEW.binding_version,NEW.created_at) THEN
        RAISE EXCEPTION 'source binding checkpoint identity is immutable' USING ERRCODE='23514';
    END IF;
    IF TG_OP='UPDATE' AND NEW.updated_at < OLD.updated_at THEN
        RAISE EXCEPTION 'source binding checkpoint updated_at cannot move backwards' USING ERRCODE='23514';
    END IF;
    IF TG_OP='INSERT' THEN
        SELECT sb.is_current,sb.status,sb.operations
          INTO binding_current,binding_status,binding_operations
          FROM source_bindings sb
         WHERE sb.tenant_id=NEW.tenant_id
           AND sb.source_id=NEW.source_id
           AND sb.binding_id=NEW.binding_id
           AND sb.version=NEW.binding_version;
        IF NOT COALESCE(binding_current,false)
           OR binding_status<>'ACTIVE'
           OR NOT (binding_operations ? 'PAGE' OR binding_operations ? 'CHANGES') THEN
            RAISE EXCEPTION 'checkpoint requires a current active stateful source binding' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER source_binding_checkpoint_guard_trigger
BEFORE INSERT OR UPDATE ON source_binding_checkpoints
FOR EACH ROW EXECUTE FUNCTION source_binding_checkpoint_guard();

COMMIT;
