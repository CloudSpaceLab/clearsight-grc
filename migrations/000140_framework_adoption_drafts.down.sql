BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM program_framework_adoptions WHERE status='DRAFT') THEN RAISE EXCEPTION 'Cannot remove existing framework adoption drafts'; END IF; END $$;
DROP INDEX program_framework_adoptions_draft_standard_uq;
ALTER TABLE program_framework_adoptions DROP CONSTRAINT program_framework_adoption_decision_ck;
ALTER TABLE program_framework_adoptions ADD CONSTRAINT program_framework_adoption_decision_ck CHECK (
  (status='PENDING' AND reviewed_by IS NULL AND reviewed_at IS NULL AND effective_from IS NULL AND effective_until IS NULL)
  OR (status='REJECTED' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND effective_from IS NULL AND effective_until IS NULL)
  OR (status='ACTIVE' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND effective_from IS NOT NULL AND effective_until IS NULL)
  OR (status IN ('SUPERSEDED','RETIRED') AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND effective_from IS NOT NULL AND effective_until IS NOT NULL AND effective_from < effective_until)
);
ALTER TABLE program_framework_adoptions DROP CONSTRAINT program_framework_adoptions_status_check;
ALTER TABLE program_framework_adoptions ADD CONSTRAINT program_framework_adoptions_status_check CHECK (status IN ('PENDING','ACTIVE','REJECTED','SUPERSEDED','RETIRED'));
COMMIT;
