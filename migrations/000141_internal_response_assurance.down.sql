BEGIN;

-- Never erase or relabel provenance to make a rollback succeed.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM capture_response_revisions WHERE achieved_assurance = 'INTERNAL_AUTHENTICATED')
       OR EXISTS (SELECT 1 FROM capture_response_workspace_edits WHERE patch->>'assurance' = 'INTERNAL_AUTHENTICATED' OR patch->>'origin' = 'INTERNAL') THEN
        RAISE EXCEPTION 'Internal response provenance exists; rollback is not safe';
    END IF;
END $$;

ALTER TABLE capture_response_revisions
    DROP CONSTRAINT capture_response_revisions_achieved_assurance_check;
ALTER TABLE capture_response_revisions
    ADD CONSTRAINT capture_response_revisions_achieved_assurance_check
    CHECK (achieved_assurance IN ('LINK_POSSESSION', 'EMAIL_VERIFIED'));

COMMIT;
