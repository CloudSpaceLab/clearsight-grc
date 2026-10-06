package workflow

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/attention"
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
		Payload:    []byte(`{"matter_id":"` + matterID + `","action_id":"30000000-0000-4000-8000-000000000001","recipient_principal_id":"40000000-0000-4000-8000-000000000001","message":"Sensitive raw update text must not be stored"}`),
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
	if record.Kind != actionUpdateNotificationKind || record.Title != "Update requested" || record.Summary != "Open Work to review the current record." {
		t.Fatalf("record = %#v", record)
	}
	for _, sensitive := range []string{"Sensitive raw update", "Provide current remediation status", "Access control exception"} {
		if strings.Contains(record.Summary, sensitive) || strings.Contains(record.Title, sensitive) {
			t.Fatalf("sensitive source text %q leaked into notification: %#v", sensitive, record)
		}
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
		Payload:    []byte(`{"matter":{"id":"` + matterID + `"},"owner_principal_id":"40000000-0000-4000-8000-000000000002","previous_owner_principal_id":"40000000-0000-4000-8000-000000000003"}`),
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

func TestInAppNotificationProjectorRendersSafeAttentionIntent(t *testing.T) {
	riskID := "20000000-0000-4000-8000-000000000010"
	principalID := "40000000-0000-4000-8000-000000000010"
	entityID := "50000000-0000-4000-8000-000000000010"
	episodeID := "60000000-0000-4000-8000-000000000010"
	sourceID := "70000000-0000-4000-8000-000000000010"
	event := workflowruntime.OutboxEvent{
		ID: "10000000-0000-4000-8000-000000000010", TenantID: "bank",
		AggregateType: attention.EpisodeAggregateType, AggregateID: episodeID, EventType: attention.EventEpisodeWorsened,
		Payload: []byte(`{"episode_id":"` + episodeID + `","legal_entity_id":"` + entityID +
			`","principal_id":"` + principalID + `","condition":"indicator_breaches","condition_state":"CRITICAL","subject_type":"RISK","subject_id":"` +
			riskID + `","source_id":"` + sourceID + `","notice_sequence":2}`),
		OccurredAt: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC),
	}
	repo := &inAppProjectionRepoStub{}
	projector := NewInAppNotificationProjector(repo, repo)
	if err := projector.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(repo.records) != 1 {
		t.Fatalf("records = %#v", repo.records)
	}
	record := repo.records[0]
	if record.Kind != "ATTENTION_INDICATOR_BREACHES_WORSENED" ||
		record.Title != "Risk indicator worsened" ||
		record.Summary != "Open the record to review current state." {
		t.Fatalf("record = %#v", record)
	}
	if record.SubjectType != "RISK" || record.SubjectID != riskID ||
		record.PrincipalID != principalID || record.LegalEntityID != entityID ||
		record.ActionPath != "#risks/"+riskID {
		t.Fatalf("target = %#v", record)
	}
	if strings.Contains(record.Title, "CRITICAL") || strings.Contains(record.Summary, sourceID) {
		t.Fatalf("raw condition/source metadata leaked into presentation: %#v", record)
	}
}

func TestInAppNotificationProjectorProjectsEscalationAssignmentSafely(t *testing.T) {
	matterID := "20000000-0000-4000-8000-000000000020"
	principalID := "40000000-0000-4000-8000-000000000020"
	event := workflowruntime.OutboxEvent{
		ID: "10000000-0000-4000-8000-000000000020", TenantID: "bank",
		AggregateType: "MATTER", AggregateID: matterID, EventType: EventMatterEscalationAssigned,
		Payload: []byte(`{"matter_id":"` + matterID + `","task_id":"30000000-0000-4000-8000-000000000020","recipient_principal_id":"` +
			principalID + `","previous_principal_id":"40000000-0000-4000-8000-000000000021","responsibility":"REVIEWER","sequence_id":"seq","step_index":1}`),
		OccurredAt: time.Date(2026, 10, 6, 3, 0, 0, 0, time.UTC),
	}
	repo := &inAppProjectionRepoStub{context: assignmentNotificationContext{
		LegalEntityID:      "50000000-0000-4000-8000-000000000020",
		CurrentPrincipalID: principalID, MatterID: matterID,
	}}
	if err := (NewInAppNotificationProjector(repo, repo)).Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(repo.records) != 1 {
		t.Fatalf("records=%#v", repo.records)
	}
	record := repo.records[0]
	if record.Kind != matterEscalationNotificationKind || record.Title != "Escalated work assigned to you" ||
		record.SubjectType != "MATTER" || record.SubjectID != matterID || record.PrincipalID != principalID {
		t.Fatalf("record=%#v", record)
	}
}

type largeAssignmentProjectionRepo struct {
	records []inAppNotificationRecord
}

func (r *largeAssignmentProjectionRepo) LoadAssignmentNotification(_ context.Context, event workflowruntime.OutboxEvent, assignment assignmentNotificationEvent) (assignmentNotificationContext, error) {
	return assignmentNotificationContext{
		LegalEntityID:      "50000000-0000-4000-8000-000000000030",
		CurrentPrincipalID: assignment.PrincipalID,
		MatterID:           event.AggregateID,
	}, nil
}

func (r *largeAssignmentProjectionRepo) StoreInAppNotification(_ context.Context, record inAppNotificationRecord) error {
	r.records = append(r.records, record)
	return nil
}

func TestInAppNotificationProjectorHandlesLargeAssignmentPopulation(t *testing.T) {
	const population = 1000
	matterID := "20000000-0000-4000-8000-000000000030"
	repo := &largeAssignmentProjectionRepo{}
	projector := NewInAppNotificationProjector(repo, repo)
	for index := 0; index < population; index++ {
		principalID := fmt.Sprintf("40000000-0000-4000-8000-%012x", index+1)
		previousID := fmt.Sprintf("41000000-0000-4000-8000-%012x", index+1)
		eventID := fmt.Sprintf("10000000-0000-4000-8000-%012x", index+1)
		event := workflowruntime.OutboxEvent{
			ID: eventID, TenantID: "bank", AggregateType: "MATTER", AggregateID: matterID,
			EventType: continuity.EventMatterOwnerChanged,
			Payload: []byte(`{"matter":{"id":"` + matterID + `"},"owner_principal_id":"` + principalID +
				`","previous_owner_principal_id":"` + previousID + `"}`),
			OccurredAt: time.Date(2026, 10, 6, 3, 0, 0, 0, time.UTC),
		}
		if err := projector.Publish(context.Background(), event); err != nil {
			t.Fatalf("assignment %d: %v", index, err)
		}
	}
	if len(repo.records) != population {
		t.Fatalf("records=%d want %d", len(repo.records), population)
	}
}
