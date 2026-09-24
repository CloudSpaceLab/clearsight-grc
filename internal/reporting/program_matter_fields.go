package reporting

// MatterReportVisibilitySQL is the report query's copy of the canonical
// continuity visibility rule. The principal is bound as $4. Every malformed
// or unsupported shape is false; a restricted Matter must have a string,
// non-empty principal allow-list and an exact allow-list match.
const MatterReportVisibilitySQL = `CASE
					WHEN NOT (a.scope ? 'access') THEN true
					WHEN jsonb_typeof(a.scope->'access')<>'string' THEN false
					WHEN upper(btrim(a.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
					WHEN upper(btrim(a.scope->>'access'))='RESTRICTED' THEN
						CASE
							WHEN jsonb_typeof(a.scope->'allowed_principal_ids')<>'array' THEN false
							ELSE
								NOT EXISTS (
									SELECT 1
									FROM jsonb_array_elements(a.scope->'allowed_principal_ids') AS entry
									WHERE jsonb_typeof(entry.value)<>'string'
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text(a.scope->'allowed_principal_ids') AS nonblank
									WHERE btrim(nonblank.value)<>''
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text(a.scope->'allowed_principal_ids') AS allowed
									WHERE btrim(allowed.value)=$4
								)
						END
					ELSE false
				END`

// MatterReportExceptionPredicateSQL defines the report's Matter exception
// dataset without a broad application-memory filter: an open EXCEPTION Matter
// or any still-open Matter whose recorded due date is past the run boundary.
const MatterReportExceptionPredicateSQL = `(
				(a.status NOT IN ('CLOSED','CANCELLED') AND a.matter_type='EXCEPTION')
				OR (a.status NOT IN ('CLOSED','CANCELLED') AND a.due_at IS NOT NULL AND a.due_at<$5::timestamptz)
			)`

// ProgramReportStatusRankSQL and MatterReportPrioritySQL are kept as named
// expressions so the page builders and their keyset predicates cannot drift.
const ProgramReportStatusRankSQL = `CASE a.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END`
const MatterReportPrioritySQL = `a.priority`
