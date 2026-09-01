BEGIN;

ALTER TABLE matter_actions
    ADD COLUMN origin_key text;

CREATE UNIQUE INDEX matter_actions_origin_key_idx
    ON matter_actions(tenant_id, matter_id, origin_key)
    WHERE origin_key IS NOT NULL;

COMMIT;
