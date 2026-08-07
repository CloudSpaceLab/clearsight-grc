BEGIN;

ALTER TABLE document_imports DROP CONSTRAINT document_imports_extraction_status_check;
ALTER TABLE document_imports DROP CONSTRAINT document_imports_analysis_status_check;

ALTER TABLE document_imports
    ADD CONSTRAINT document_imports_extraction_status_check CHECK (extraction_status IN ('PENDING','EXTRACTED','UNSUPPORTED','FAILED')),
    ADD CONSTRAINT document_imports_analysis_status_check CHECK (analysis_status IN ('PENDING','REVIEW_REQUIRED','NO_PROPOSALS','UNAVAILABLE')),
    ADD COLUMN sections_total integer NOT NULL DEFAULT 0 CHECK (sections_total >= 0),
    ADD COLUMN sections_omitted integer NOT NULL DEFAULT 0 CHECK (sections_omitted >= 0),
    ADD COLUMN proposals_total integer NOT NULL DEFAULT 0 CHECK (proposals_total >= 0),
    ADD COLUMN proposals_omitted integer NOT NULL DEFAULT 0 CHECK (proposals_omitted >= 0),
    ADD COLUMN content_truncated boolean NOT NULL DEFAULT false,
    ADD COLUMN processed_at timestamptz;

UPDATE document_imports
SET sections_total=jsonb_array_length(sections),
    proposals_total=jsonb_array_length(proposals)
WHERE sections_total=0 AND proposals_total=0;

CREATE INDEX document_imports_processing_idx
    ON document_imports(tenant_id,created_at,id)
    WHERE extraction_status='PENDING' OR analysis_status='PENDING';

COMMIT;
