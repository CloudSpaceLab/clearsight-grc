//go:build postgres && postgresintegration

package evidence

import (
	"os"
	"testing"
)

func TestInternalWorkspaceAssuranceMigrationCannotEraseProvenance(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()
	// Session-local tables exercise the actual SQL without changing the shared schema.
	_, err = connection.Exec(ctx, `CREATE TEMP TABLE capture_response_revisions (
		achieved_assurance text NOT NULL CONSTRAINT capture_response_revisions_achieved_assurance_check
		CHECK (achieved_assurance IN ('LINK_POSSESSION','EMAIL_VERIFIED'))
	); CREATE TEMP TABLE capture_response_workspace_edits (patch jsonb NOT NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = connection.Exec(ctx, `DROP TABLE IF EXISTS pg_temp.capture_response_workspace_edits, pg_temp.capture_response_revisions`)
	}()
	up, err := os.ReadFile("../../migrations/000141_internal_response_assurance.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000141_internal_response_assurance.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, string(up)); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if _, err = connection.Exec(ctx, `INSERT INTO capture_response_revisions VALUES ('LINK_POSSESSION'),('EMAIL_VERIFIED'),('INTERNAL_AUTHENTICATED')`); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, string(down)); err == nil {
		t.Fatal("rollback discarded internal revision provenance")
	}
	if _, err = connection.Exec(ctx, `ROLLBACK`); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, `DELETE FROM capture_response_revisions WHERE achieved_assurance='INTERNAL_AUTHENTICATED'; INSERT INTO capture_response_workspace_edits VALUES ('{"origin":"INTERNAL","assurance":"INTERNAL_AUTHENTICATED"}'::jsonb)`); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, string(down)); err == nil {
		t.Fatal("rollback discarded internal edit provenance")
	}
	if _, err = connection.Exec(ctx, `ROLLBACK`); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, `DELETE FROM capture_response_workspace_edits`); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, string(down)); err != nil {
		t.Fatalf("safe rollback: %v", err)
	}
	if _, err = connection.Exec(ctx, `INSERT INTO capture_response_revisions VALUES ('INTERNAL_AUTHENTICATED')`); err == nil {
		t.Fatal("rollback left internal assurance accepted")
	}
}
