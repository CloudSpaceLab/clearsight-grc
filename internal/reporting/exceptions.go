package reporting

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

// ExceptionBlocker describes one missing fact, in the same words the register
// shows the operator, so a report row and a register row read identically.
type ExceptionBlocker struct {
	Field   string `json:"field"`
	Missing string `json:"missing"`
}

// ExceptionBlockers returns the missing facts for one activity, delegating to
// the register so the two surfaces cannot diverge.
func ExceptionBlockers(activity ropa.ProcessingActivity) []ExceptionBlocker {
	names := ropa.ClosureBlockersForTest(activity)
	blockers := make([]ExceptionBlocker, 0, len(names))
	for _, name := range names {
		blockers = append(blockers, ExceptionBlocker{Field: blockerField(name), Missing: name})
	}
	return blockers
}

func blockerField(name string) string {
	switch name {
	case "lawful basis":
		return string(ReportFieldMissingBasis)
	case "named owner":
		return string(ReportFieldMissingOwner)
	case "data subject category":
		return string(ReportFieldMissingSubjects)
	case "completed review":
		return string(ReportFieldReviewOverdue)
	default:
		return strings.ReplaceAll(name, " ", "_")
	}
}

// ExceptionPredicate is the in-process form of the SQL predicate below. The
// parity test pins the two together.
func ExceptionPredicate(activity ropa.ProcessingActivity) bool {
	return len(ropa.ClosureBlockersForTest(activity)) > 0
}

// ExceptionPredicateSQL is the database form. Each clause mirrors one branch of
// ropa.closureBlockers exactly. The whitespace expression corresponds to the
// register's strings.TrimSpace checks; review outcomes are deliberately not
// trimmed because the register accepts only the exact recorded vocabulary.
const ExceptionPredicateSQL = `(
  NULLIF(regexp_replace(a.lawful_basis, '^[[:space:]]+|[[:space:]]+$', '', 'g'), '') IS NULL
  OR a.owner_principal_id IS NULL
  OR NULLIF(regexp_replace(a.data_subject_categories, '^[[:space:]]+|[[:space:]]+$', '', 'g'), '') IS NULL
  OR NOT EXISTS (
       SELECT 1 FROM ropa_processing_activity_reviews r
       WHERE r.tenant_id=a.tenant_id AND r.legal_entity_id=a.legal_entity_id AND r.activity_id=a.id
         AND r.completed_at IS NOT NULL AND r.outcome IN ('CONFIRMED','REVISED'))
)`

// ExceptionColumnSQL lists, per row, which facts are missing. The expressions
// match ExceptionPredicateSQL one for one.
const ExceptionColumnSQL = `ARRAY_REMOVE(ARRAY[
  CASE WHEN NULLIF(regexp_replace(a.lawful_basis, '^[[:space:]]+|[[:space:]]+$', '', 'g'), '') IS NULL THEN 'Lawful basis not recorded' END,
  CASE WHEN a.owner_principal_id IS NULL THEN 'Owner not recorded' END,
  CASE WHEN NULLIF(regexp_replace(a.data_subject_categories, '^[[:space:]]+|[[:space:]]+$', '', 'g'), '') IS NULL THEN 'Data subject categories not recorded' END,
  CASE WHEN NOT EXISTS (
       SELECT 1 FROM ropa_processing_activity_reviews r
       WHERE r.tenant_id=a.tenant_id AND r.legal_entity_id=a.legal_entity_id AND r.activity_id=a.id
         AND r.completed_at IS NOT NULL AND r.outcome IN ('CONFIRMED','REVISED'))
       THEN 'No completed review' END], NULL)`
