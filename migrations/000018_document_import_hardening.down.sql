BEGIN;

DROP INDEX IF EXISTS document_imports_processing_idx;

UPDATE document_imports
SET extraction_status='FAILED',
    analysis_status='UNAVAILABLE',
    extraction_method=CASE WHEN extraction_status='PENDING' THEN 'ROLLED_BACK_PENDING' ELSE extraction_method END,
    analysis_method=CASE WHEN analysis_status='PENDING' THEN 'ROLLED_BACK_PENDING' ELSE analysis_method END
WHERE extraction_status='PENDING' OR analysis_status='PENDING';

ALTER TABLE document_imports DROP CONSTRAINT document_imports_extraction_status_check;
ALTER TABLE document_imports DROP CONSTRAINT document_imports_analysis_status_check;
ALTER TABLE document_imports
    ADD CONSTRAINT document_imports_extraction_status_check CHECK (extraction_status IN ('EXTRACTED','UNSUPPORTED','FAILED')),
    ADD CONSTRAINT document_imports_analysis_status_check CHECK (analysis_status IN ('REVIEW_REQUIRED','NO_PROPOSALS','UNAVAILABLE')),
    DROP COLUMN processed_at,
    DROP COLUMN content_truncated,
    DROP COLUMN proposals_omitted,
    DROP COLUMN proposals_total,
    DROP COLUMN sections_omitted,
    DROP COLUMN sections_total;

COMMIT;
