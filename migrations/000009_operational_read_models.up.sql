BEGIN;

ALTER TABLE programs
    ADD COLUMN search_document tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple'::regconfig,
            coalesce(code,'') || ' ' ||
            coalesce(name,'') || ' ' ||
            coalesce(owning_function,'') || ' ' ||
            coalesce(jurisdiction,''))
    ) STORED;

ALTER TABLE matters
    ADD COLUMN search_document tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple'::regconfig,
            coalesce(reference,'') || ' ' ||
            coalesce(title,'') || ' ' ||
            coalesce(summary,'') || ' ' ||
            coalesce(matter_type,''))
    ) STORED;

CREATE INDEX programs_summary_order_idx
    ON programs(
        tenant_id,
        (CASE status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END),
        updated_at DESC,
        id DESC
    );
CREATE INDEX programs_search_idx ON programs USING gin(search_document);

CREATE INDEX matters_summary_order_idx
    ON matters(tenant_id,priority DESC,updated_at DESC,id DESC);
CREATE INDEX matters_open_summary_order_idx
    ON matters(tenant_id,priority DESC,updated_at DESC,id DESC)
    WHERE status NOT IN ('CLOSED','CANCELLED');
CREATE INDEX matters_search_idx ON matters USING gin(search_document);

COMMIT;
