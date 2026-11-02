//go:build postgres

package workflow

import (
	"fmt"
	"time"
)

// postgresWorkPagePredicate is shared by the canonical workflow and document
// proposal reads. Argument positions are static; no request data becomes SQL.
func postgresWorkPagePredicate(start int) string {
	fence, enabled, due, updated, id := start, start+1, start+2, start+3, start+4
	return fmt.Sprintf(`
	  AND ($%d::timestamptz IS NULL OR wt.updated_at <= $%d::timestamptz)
	  AND (NOT $%d::boolean OR (
	    ($%d::timestamptz IS NOT NULL AND (
	      wt.due_at IS NULL OR wt.due_at > $%d::timestamptz OR
	      (wt.due_at = $%d::timestamptz AND
	        (wt.updated_at < $%d::timestamptz OR
	          (wt.updated_at = $%d::timestamptz AND wt.id > NULLIF($%d,'')::uuid)))
	    ))
	    OR ($%d::timestamptz IS NULL AND wt.due_at IS NULL AND
	      (wt.updated_at < $%d::timestamptz OR
	        (wt.updated_at = $%d::timestamptz AND wt.id > NULLIF($%d,'')::uuid)))
	  ))`,
		fence, fence, enabled, due, due, due, updated, updated, id,
		due, updated, updated, id)
}

func postgresWorkCursorArgs(filter ListFilter) (bool, *time.Time, *time.Time, string) {
	if filter.PageAfter == nil {
		return false, nil, nil, ""
	}
	return true, filter.PageAfter.DueAt, &filter.PageAfter.UpdatedAt, filter.PageAfter.ID
}
