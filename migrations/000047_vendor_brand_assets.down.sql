BEGIN;

-- Append-only third_party_events/outbox rows and the VENDOR_BRAND vocabulary
-- are intentionally retained. The tenant guard references the vendor stream,
-- not the asset/job tables removed below, so retained VENDOR_BRAND event history
-- remains valid and reconstructable after this structural rollback.
DROP TABLE third_party_vendor_brand_jobs;
DROP TABLE third_party_vendor_brand_assets;
ALTER TABLE third_parties
    DROP CONSTRAINT third_parties_website_domain_check,
    DROP COLUMN website_domain;
DROP FUNCTION third_party_website_domain_valid(text);

COMMIT;
