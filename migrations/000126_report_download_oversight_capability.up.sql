BEGIN;

UPDATE role_templates
SET capabilities = (
    SELECT ARRAY(
        SELECT DISTINCT value
        FROM unnest(capabilities || ARRAY['REPORT_DOWNLOAD']) value
        ORDER BY value
    )
)
WHERE valid_until IS NULL
  AND code IN ('CRO','CCO','CISO','EXECUTIVE','GRC_ADMIN','INTERNAL_AUDITOR');

COMMIT;
