BEGIN;

-- Brand events and their vocabulary remain append-only. Structural rollback
-- removes only the command ledgers introduced by this migration.
DROP TABLE third_party_vendor_brand_command_receipts;
DROP TABLE third_party_vendor_brand_upload_reservations;
ALTER TABLE third_party_vendor_brand_assets DROP COLUMN asset_token;

COMMIT;
