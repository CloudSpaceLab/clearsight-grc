//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAssignmentNotificationDatabaseConstraintsAreTenantComposite(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid='staff_assignment_notification_deliveries'::regclass
		ORDER BY conname`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	definitions := ""
	for rows.Next() {
		var definition string
		if err := rows.Scan(&definition); err != nil {
			t.Fatal(err)
		}
		definitions += strings.ToLower(definition) + "\n"
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"unique (tenant_id, outbox_event_id, principal_id, notification_kind)",
		"foreign key (legal_entity_id, tenant_id) references legal_entities(id, tenant_id)",
		"foreign key (outbox_event_id, tenant_id) references outbox_events(id, tenant_id)",
		"foreign key (principal_id, tenant_id) references principals(id, tenant_id)",
	} {
		if !strings.Contains(definitions, required) {
			t.Fatalf("database schema is missing %q; constraints:\n%s", required, definitions)
		}
	}
}
