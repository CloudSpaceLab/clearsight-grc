DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM third_party_response_application_receipts LIMIT 1) THEN
        RAISE EXCEPTION 'cannot remove vendor refresh schema while response application receipts exist';
    END IF;
    IF EXISTS (SELECT 1 FROM third_party_assessments WHERE scope_kind<>'FULL' OR selected_field_ids<>'[]'::jsonb LIMIT 1) THEN
        RAISE EXCEPTION 'cannot remove vendor refresh schema while focused assessments exist';
    END IF;
    IF EXISTS (SELECT 1 FROM third_party_documents WHERE status='SUPERSEDED' OR supersedes_document_id IS NOT NULL LIMIT 1) THEN
        RAISE EXCEPTION 'cannot remove vendor refresh schema while document supersession history exists';
    END IF;
END $$;

DROP TABLE third_party_response_application_receipts;
DROP INDEX third_party_documents_due_idx;
DROP INDEX third_party_documents_current_type_idx;
ALTER TABLE third_party_documents
    DROP CONSTRAINT third_party_documents_validation_state_check,
    DROP CONSTRAINT third_party_documents_supersedes_fk,
    DROP COLUMN supersedes_document_id,
    DROP CONSTRAINT third_party_documents_status_check,
    ADD CONSTRAINT third_party_documents_status_check CHECK (status IN ('SUBMITTED','VALIDATED','REJECTED','EXPIRED')),
    ADD CONSTRAINT third_party_documents_validation_state_check CHECK (
        (status IN ('VALIDATED','REJECTED') AND validated_by_principal_id IS NOT NULL AND validated_at IS NOT NULL)
        OR (status IN ('SUBMITTED','EXPIRED') AND validated_by_principal_id IS NULL AND validated_at IS NULL)
    );
ALTER TABLE third_party_assessments
    DROP CONSTRAINT third_party_assessments_selected_fields_check,
    DROP CONSTRAINT third_party_assessments_scope_version_check,
    DROP CONSTRAINT third_party_assessments_scope_kind_check,
    DROP COLUMN selected_field_ids,
    DROP COLUMN scope_version,
    DROP COLUMN scope_kind;
