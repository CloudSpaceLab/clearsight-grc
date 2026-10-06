BEGIN;

UPDATE role_templates
SET capabilities = array_remove(capabilities, 'REPORT_DOWNLOAD')
WHERE valid_until IS NULL
  AND code IN ('CCO','GRC_ADMIN');

COMMIT;
