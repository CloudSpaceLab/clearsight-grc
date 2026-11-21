BEGIN;
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM capture_response_reviews WHERE authority_source_type IS NOT NULL OR authority_source_id IS NOT NULL OR authority_rule_id IS NOT NULL OR authority_policy_version IS NOT NULL) THEN
  RAISE EXCEPTION 'cannot remove recorded response review authority receipts';
 END IF;
END $$;
ALTER TABLE capture_response_reviews DROP CONSTRAINT capture_response_reviews_authority_receipt_check,
 DROP COLUMN authority_source_type, DROP COLUMN authority_source_id, DROP COLUMN authority_rule_id, DROP COLUMN authority_policy_version;
COMMIT;
