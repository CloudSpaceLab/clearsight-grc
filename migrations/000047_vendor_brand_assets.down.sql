BEGIN;

DROP TABLE third_party_vendor_brand_jobs;
DROP TABLE third_party_vendor_brand_assets;
ALTER TABLE third_parties
    DROP CONSTRAINT third_parties_website_domain_check,
    DROP COLUMN website_domain;
DROP FUNCTION third_party_website_domain_valid(text);

COMMIT;
