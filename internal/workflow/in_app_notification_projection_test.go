package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type inAppProjectionRepoStub struct {
	context assignmentNotificationContext
	records []inAppNotificationRecord
}

func (s *inAppProjectionRepoStub) LoadAssignmentNotification(context.Context, workflowruntime.OutboxEvent, assignmentNotificationEvent) (assignmentNotificationContext, error) {
	return s.context, nil
}

func (s *inAppProjectionRepoStub) StoreInAppNotification(_ context.Context, record inAppNotificationRecord) error {
	s.records = append(s.records, record)
	return nil
}

func TestInAppNotificationProjectorDoesNotPersistRawUpdateMessage(t *testing.T) {
	matterID := "20000000-0000-4000-8000-000000000001"
	event := workflowruntime.OutboxEvent{
		ID: "10000000-0000-4000-8000-000000000001", TenantID: "bank",
		AggregateType: "MATTER", AggregateID: matterID, EventType: continuity.EventMatterActionUpdateRequested,
		Payload: []byte(`{"matter_id":"` + matterID + `","action_id":"30000000-0000-4000-8000-000000000001","recipient_principal_id":"40000000-0000-4000-8000-000000000001","message":"Sensitive raw update text must not be stored"}`),
		OccurredAt: time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC),
	}
	repo := &inAppProjectionRepoStub{context: assignmentNotificationContext{
		LegalEntityID:      "50000000-0000-4000-8000-000000000001",
		CurrentPrincipalID: "40000000-0000-4000-8000-000000000001",
		MatterID:           matterID, MatterTitle: "Access control exception", WorkTitle: "Provide current remediation status",
	}}
	projector := NewInAppNotificationProjector(repo, repo)
	if err := projector.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(repo.records) != 1 {
		t.Fatalf("records = %#v", repo.records)
	}
	record := repo.records[0]
	if record.Kind != actionUpdateNotificationKind || record.Title != "Update requested" || record.Summary != "Provide current remediation status" {
		t.Fatalf("record = %#v", record)
	}
	if strings.Contains(record.Summary, "Sensitive raw update") || strings.Contains(record.Title, "Sensitive raw update") {
		t.Fatalf("raw request text leaked into notification: %#v", record)
	}
	if record.SubjectType != "MATTER" || record.SubjectID != matterID || record.ActionPath != "#work/matters/"+matterID {
		t.Fatalf("target = %#v", record)
	}
}

func TestInAppNotificationProjectorSkipsSupersededAssignment(t *testing.T) {
	matterID := "20000000-0000-4000-8000-000000000002"
	event := workflowruntime.OutboxEvent{
		ID: "10000000-0000-4000-8000-000000000002", TenantID: "bank",
		AggregateType: "MATTER", AggregateID: matterID, EventType: continuity.EventMatterOwnerChanged,
		Payload: []byte(`{"matter":{"id":"` + matterID + `"},"owner_principal_id":"40000000-0000-4000-8000-000000000002","previous_owner_principal_id":"40000000-0000-4000-8000-000000000003"}`),
		OccurredAt: time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC),
	}
	repo := &inAppProjectionRepoStub{context: assignmentNotificationContext{
		LegalEntityID:      "50000000-0000-4000-8000-000000000001",
		CurrentPrincipalID: "40000000-0000-4000-8000-000000000099",
		MatterID:           matterID, MatterTitle: "Superseded assignment",
	}}
	if err := (NewInAppNotificationProjector(repo, repo)).Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(repo.records) != 0 {
		t.Fatalf("superseded assignment produced notification: %#v", repo.records)
	}
}
