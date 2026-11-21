BEGIN;

-- External access sessions keep their existing, narrower assurance constraint.
ALTER TABLE capture_response_revisions
    DROP CONSTRAINT capture_response_revisions_achieved_assurance_check;
ALTER TABLE capture_response_revisions
    ADD CONSTRAINT capture_response_revisions_achieved_assurance_check
    CHECK (achieved_assurance IN ('LINK_POSSESSION', 'EMAIL_VERIFIED', 'INTERNAL_AUTHENTICATED'));

COMMIT;
