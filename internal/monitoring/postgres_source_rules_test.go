//go:build postgres

package monitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

var errSourceRulesCapture = errors.New("stop after query capture")

type sourceRulesCaptureDB struct {
	args []any
}

func (db *sourceRulesCaptureDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	db.args = append([]any(nil), args...)
	return sourceRulesCaptureRow{}
}

type sourceRulesCaptureRow struct{}

func (sourceRulesCaptureRow) Scan(...any) error { return errSourceRulesCapture }

func TestInsertCheckRevisionSerializesNilSourceRulesAsEmptyArray(t *testing.T) {
	db := &sourceRulesCaptureDB{}

	_, err := insertCheckRevision(context.Background(), db, MonitoringCheck{})
	if !errors.Is(err, errSourceRulesCapture) {
		t.Fatalf("insertCheckRevision error = %v, want capture sentinel", err)
	}
	if len(db.args) != 32 {
		t.Fatalf("captured %d query arguments, want 32", len(db.args))
	}

	rules, ok := db.args[14].([]byte)
	if !ok {
		t.Fatalf("source_rules argument type = %T, want []byte", db.args[14])
	}
	if got := string(rules); got != "[]" {
		t.Fatalf("source_rules argument = %s, want []", got)
	}
}
