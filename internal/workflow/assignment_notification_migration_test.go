package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestAssignmentNotificationMigrationScopesEveryReferenceAndReceiptToTenant(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000061_staff_assignment_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(payload))
	for _, required := range []string{
		"UNIQUE (TENANT_ID, OUTBOX_EVENT_ID, PRINCIPAL_ID, NOTIFICATION_KIND)",
		"FOREIGN KEY (LEGAL_ENTITY_ID, TENANT_ID) REFERENCES LEGAL_ENTITIES(ID, TENANT_ID)",
		"FOREIGN KEY (OUTBOX_EVENT_ID, TENANT_ID) REFERENCES OUTBOX_EVENTS(ID, TENANT_ID)",
		"FOREIGN KEY (PRINCIPAL_ID, TENANT_ID) REFERENCES PRINCIPALS(ID, TENANT_ID)",
		"DELIVERY_STARTED",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("assignment notification migration is missing %q", required)
		}
	}
	for _, prohibited := range []string{
		"LEGAL_ENTITY_ID UUID NOT NULL REFERENCES LEGAL_ENTITIES(ID)",
		"OUTBOX_EVENT_ID UUID NOT NULL REFERENCES OUTBOX_EVENTS(ID)",
		"PRINCIPAL_ID UUID NOT NULL REFERENCES PRINCIPALS(ID)",
		"UNIQUE (OUTBOX_EVENT_ID, PRINCIPAL_ID, NOTIFICATION_KIND)",
	} {
		if strings.Contains(sql, prohibited) {
			t.Fatalf("assignment notification migration contains tenant-unsafe constraint %q", prohibited)
		}
	}
}
