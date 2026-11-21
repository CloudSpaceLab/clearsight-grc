//go:build postgres

package workflow

import "fmt"

// The Program must exist in the request's tenant and legal entity. A reviewer
// or respondent assignment does not override a restricted Program scope.
func programEvidenceVisibilitySQL(principal string) string {
	return fmt.Sprintf(`CASE
 WHEN ep.id IS NULL THEN false
 WHEN NOT (ep.scope ? 'access') THEN true
 WHEN upper(btrim(COALESCE(ep.scope->>'access',''))) IN ('PUBLIC','INTERNAL') THEN true
 WHEN upper(btrim(COALESCE(ep.scope->>'access','')))='RESTRICTED' THEN
   CASE WHEN jsonb_typeof(ep.scope->'allowed_principal_ids')='array' THEN
     NOT EXISTS (SELECT 1 FROM jsonb_array_elements(ep.scope->'allowed_principal_ids') entry(value) WHERE jsonb_typeof(entry.value)<>'string')
     AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(ep.scope->'allowed_principal_ids') allowed(value) WHERE btrim(allowed.value)=%s)
   ELSE false END
 ELSE false END`, principal)
}
