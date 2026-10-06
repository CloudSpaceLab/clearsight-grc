package httpapi

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
)

type notificationDeliveryHistoryItem struct {
	ID         string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`
	EventType  string    `json:"event_type"`
	Action     string    `json:"action"`
	Outcome    string    `json:"outcome"`
}

type operationalLossRead struct {
	oploss.Aggregate
	OwnerDisplayName            string                            `json:"owner_display_name,omitempty"`
	NotificationHistory         []notificationDeliveryHistoryItem `json:"notification_history"`
	NotificationHistoryComplete bool                              `json:"notification_history_complete"`
}

func (a *API) notificationDeliveryHistory(ctx context.Context, actor identity.Actor, objectType, objectID string) ([]notificationDeliveryHistoryItem, bool) {
	if a == nil || a.deps.Activity == nil {
		return []notificationDeliveryHistoryItem{}, false
	}
	page, err := a.deps.Activity.List(ctx, activity.Query{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		ObjectType: objectType, ObjectID: objectID,
		Source: activity.SourceNotificationEmailDelivery, Limit: 20,
	})
	if err != nil {
		return []notificationDeliveryHistoryItem{}, false
	}
	items := make([]notificationDeliveryHistoryItem, 0, len(page.Items))
	for _, event := range page.Items {
		items = append(items, notificationDeliveryHistoryItem{
			ID: event.ID, OccurredAt: event.OccurredAt, EventType: event.EventType,
			Action: event.Action, Outcome: event.Outcome,
		})
	}
	return items, true
}
