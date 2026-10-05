package workflow

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/attention"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type InAppNotificationProjector struct {
	loader assignmentNotificationContextLoader
	writer inAppNotificationWriter
}

func NewInAppNotificationProjector(loader assignmentNotificationContextLoader, writer inAppNotificationWriter) *InAppNotificationProjector {
	return &InAppNotificationProjector{loader: loader, writer: writer}
}

func (p *InAppNotificationProjector) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if p == nil || p.loader == nil || p.writer == nil {
		return ErrNotificationUnavailable
	}
	intent, attentionRelevant, err := attention.DecodeIntent(event)
	if err != nil {
		return err
	}
	if attentionRelevant {
		return p.projectAttention(ctx, event, intent)
	}
	assignments, relevant, err := decodeAssignmentNotificationEvent(event)
	if err != nil {
		return err
	}
	if !relevant {
		return nil
	}
	for _, assignment := range assignments {
		if err := p.project(ctx, event, assignment); err != nil {
			return err
		}
	}
	return nil
}

func (p *InAppNotificationProjector) project(ctx context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent) error {
	value, err := p.loader.LoadAssignmentNotification(ctx, event, assignment)
	if err != nil {
		return err
	}
	if !assignment.CommentMentioned && strings.TrimSpace(value.CurrentPrincipalID) != strings.TrimSpace(assignment.PrincipalID) {
		return nil
	}
	title, summary, err := inAppNotificationPresentation(assignment)
	if err != nil {
		return err
	}
	if !validNotificationUUID(value.MatterID) || !validNotificationUUID(event.ID) {
		return fmt.Errorf("in-app notification event identity is invalid")
	}
	return p.writer.StoreInAppNotification(ctx, inAppNotificationRecord{
		TenantID: event.TenantID, LegalEntityID: value.LegalEntityID, PrincipalID: assignment.PrincipalID,
		OutboxEventID: event.ID, Kind: assignment.NotificationKind,
		Title: title, Summary: summary, SubjectType: "MATTER", SubjectID: value.MatterID,
		ActionPath: "#work/matters/" + url.PathEscape(value.MatterID), OccurredAt: event.OccurredAt,
	})
}

func (p *InAppNotificationProjector) projectAttention(ctx context.Context, event workflowruntime.OutboxEvent, intent attention.Intent) error {
	title, summary, kind, err := attentionNotificationPresentation(event.EventType, intent.Condition)
	if err != nil {
		return err
	}
	actionPath := ""
	switch intent.SubjectType {
	case "RISK":
		actionPath = "#risks/" + url.PathEscape(intent.SubjectID)
	case "LOSS":
		actionPath = "#losses/" + url.PathEscape(intent.SubjectID)
	default:
		return fmt.Errorf("unsupported attention subject type %q", intent.SubjectType)
	}
	if !validNotificationUUID(intent.SubjectID) || !validNotificationUUID(intent.PrincipalID) ||
		!validNotificationUUID(intent.LegalEntityID) || !validNotificationUUID(event.ID) {
		return fmt.Errorf("attention notification event identity is invalid")
	}
	return p.writer.StoreInAppNotification(ctx, inAppNotificationRecord{
		TenantID: event.TenantID, LegalEntityID: intent.LegalEntityID, PrincipalID: intent.PrincipalID,
		OutboxEventID: event.ID, Kind: kind, Title: title, Summary: summary,
		SubjectType: intent.SubjectType, SubjectID: intent.SubjectID, ActionPath: actionPath, OccurredAt: event.OccurredAt,
	})
}

func attentionNotificationPresentation(eventType, condition string) (string, string, string, error) {
	transition := ""
	switch eventType {
	case attention.EventEpisodeOpened:
		transition = "OPENED"
	case attention.EventEpisodeWorsened:
		transition = "WORSENED"
	case attention.EventEpisodeCleared:
		transition = "CLEARED"
	default:
		return "", "", "", fmt.Errorf("unsupported attention transition %q", eventType)
	}
	var title string
	switch condition {
	case "risks_outside_appetite":
		switch transition {
		case "OPENED":
			title = "Risk outside appetite"
		case "CLEARED":
			title = "Risk returned within appetite"
		default:
			title = "Risk exposure worsened"
		}
	case "indicator_breaches":
		switch transition {
		case "OPENED":
			title = "Risk indicator breached"
		case "WORSENED":
			title = "Risk indicator worsened"
		case "CLEARED":
			title = "Risk indicator cleared"
		}
	case "assurance_failures":
		if transition == "CLEARED" {
			title = "Assurance failure cleared"
		} else {
			title = "Assurance failed"
		}
	case "losses_without_issue":
		if transition == "CLEARED" {
			title = "Loss issue linked"
		} else {
			title = "Loss has no linked issue"
		}
	default:
		return "", "", "", fmt.Errorf("unsupported attention condition %q", condition)
	}
	kind := "ATTENTION_" + strings.ToUpper(condition) + "_" + transition
	if len(kind) > 64 {
		return "", "", "", fmt.Errorf("attention notification kind is too long")
	}
	return title, "Open the record to review current state.", kind, nil
}

func inAppNotificationPresentation(assignment assignmentNotificationEvent) (string, string, error) {
	const summary = "Open Work to review the current record."
	switch assignment.NotificationKind {
	case matterOwnerNotificationKind:
		return "Issue assigned to you", summary, nil
	case actionPerformerNotificationKind:
		return "Action assigned to you", summary, nil
	case actionUpdateNotificationKind:
		return "Update requested", summary, nil
	case commentMentionNotificationKind:
		return "You were mentioned", summary, nil
	default:
		return "", "", fmt.Errorf("unsupported in-app notification kind %q", assignment.NotificationKind)
	}
}

var _ workflowruntime.Publisher = (*InAppNotificationProjector)(nil)
