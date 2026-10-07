BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM organization_metric_daily_sources) THEN
        RAISE EXCEPTION 'Organization metric daily history exists; refusing to erase it';
    END IF;
END;
$$;

DROP TRIGGER organization_metric_daily_buckets_immutable ON organization_metric_daily_buckets;
DROP TRIGGER organization_metric_daily_sources_immutable ON organization_metric_daily_sources;
DROP FUNCTION prevent_organization_metric_daily_mutation();
DROP TABLE organization_metric_daily_buckets;
DROP TABLE organization_metric_daily_sources;

COMMIT;
