BEGIN;
ALTER TABLE capture_response_reviews
 ADD COLUMN authority_source_type text,
 ADD COLUMN authority_source_id uuid,
 ADD COLUMN authority_rule_id text,
 ADD COLUMN authority_policy_version text,
 ADD CONSTRAINT capture_response_reviews_authority_receipt_check CHECK (
   (authority_source_type IS NULL AND authority_source_id IS NULL AND authority_rule_id IS NULL AND authority_policy_version IS NULL)
   OR (authority_source_type IS NOT NULL AND authority_source_type IN ('POLICY','ASSIGNMENT') AND authority_source_id IS NOT NULL AND authority_rule_id IS NOT NULL AND length(btrim(authority_rule_id)) > 0 AND authority_policy_version IS NOT NULL AND length(btrim(authority_policy_version)) > 0)
 );
COMMIT;
