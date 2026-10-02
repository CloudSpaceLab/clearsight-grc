package workflow

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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
