package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestInAppNotificationMigrationKeepsDeliveryMetadataActorScoped(t *testing.T) {
	payload, err := os.ReadFile("../../migrations/000098_in_app_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(payload))
	for _, required := range []string{
		"UNIQUE (TENANT_ID, OUTBOX_EVENT_ID, PRINCIPAL_ID, NOTIFICATION_KIND)",
		"FOREIGN KEY (LEGAL_ENTITY_ID, TENANT_ID) REFERENCES LEGAL_ENTITIES(ID, TENANT_ID)",
		"FOREIGN KEY (OUTBOX_EVENT_ID, TENANT_ID) REFERENCES OUTBOX_EVENTS(ID, TENANT_ID)",
		"FOREIGN KEY (PRINCIPAL_ID, TENANT_ID) REFERENCES PRINCIPALS(ID, TENANT_ID)",
		"IN_APP_NOTIFICATIONS_ACTOR_RECENT_IDX",
		"IN_APP_NOTIFICATIONS_ACTOR_UNREAD_IDX",
		"WHERE READ_AT IS NULL",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("in-app notification migration is missing %q", required)
		}
	}
	for _, prohibited := range []string{
		"RECIPIENT_ADDRESS",
		"EMAIL_ADDRESS",
		"MESSAGE_BODY",
		"EVIDENCE_PAYLOAD",
		"UNIQUE (OUTBOX_EVENT_ID, PRINCIPAL_ID, NOTIFICATION_KIND)",
	} {
		if strings.Contains(sql, prohibited) {
			t.Fatalf("in-app notification migration contains unsafe or unscoped field %q", prohibited)
		}
	}
}
