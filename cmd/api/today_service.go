package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const todayItemLimit = 50

type todayIdentityStatus interface {
	OperationalStatus(context.Context, string, int) (access.OperationalStatus, error)
}

type todayJobStatus interface {
	Snapshot(context.Context, string, int) (operations.Snapshot, error)
}

// actorTodayService uses the same stored Workflow projection in every runtime
// mode. Demo data must be seeded through the normal repositories; it must not
// switch Today to a separate sample journey or static item source.
func actorTodayService(workflowService *workflow.Service, identityStatus todayIdentityStatus, jobStatus todayJobStatus) *today.Service {
	return today.NewDynamicService(func(loadCtx context.Context, actor identity.Actor) ([]today.AttentionItem, error) {
		assigned, err := workflowService.List(loadCtx, workflow.ListFilter{
			TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
			ActiveOnly: true, VisibleActorWorkOnly: true, Limit: todayItemLimit,
		})
		if err != nil {
			return nil, err
		}
		items := today.FromWorkflowTasksForActor(assigned, actor.PrincipalID)

		if identityStatus != nil && identity.HasPermission(actor, identity.PermissionIdentityRead) {
			overview, overviewErr := identityStatus.OperationalStatus(loadCtx, actor.TenantID, todayItemLimit)
			if overviewErr != nil {
				return nil, overviewErr
			}
			items = append(items, identityAdministrationItems(overview)...)
		}
		if jobStatus != nil && identity.HasPermission(actor, identity.PermissionPlatformOperationsRead) {
			snapshot, snapshotErr := jobStatus.Snapshot(loadCtx, actor.TenantID, todayItemLimit)
			if snapshotErr != nil {
				return nil, snapshotErr
			}
			items = append(items, failedJobItems(snapshot)...)
		}
		return items, nil
	})
}

func identityAdministrationItems(overview access.OperationalStatus) []today.AttentionItem {
	items := make([]today.AttentionItem, 0, len(overview.SourceExceptions)+2)
	for _, source := range overview.SourceExceptions {
		code := strings.TrimSpace(source.Code)
		if code == "" {
			code = "directory"
		}
		items = append(items, today.AttentionItem{
			ID: "admin-directory-" + attentionKey(source.ID, code), Type: "IDENTITY_SOURCE",
			Title:  "Review " + code + " directory source",
			WhyNow: fmt.Sprintf("The stored directory source status is %s; review it before relying on new identities or group changes.", humanStatus(source.Status)),
			Scope:  "Identity and access", State: humanStatus(source.Status), Evidence: "Stored directory source status",
			Owner: "System administration", PrimaryAction: "Review directory source", ActionTargetType: "CONFIGURE", ActionTargetID: "access",
			InterventionClass: today.InterventionEscalation,
		})
	}
	if count := overview.Escalation.Unresolved24h; count > 0 {
		items = append(items, today.AttentionItem{
			ID: "admin-escalations-unresolved", Type: "ROUTING_EXCEPTION", Title: fmt.Sprintf("Resolve %d routing failures", count),
			WhyNow: fmt.Sprintf("%d escalation routes have remained unresolved for more than 24 hours and may leave assigned work without a valid handoff.", count),
			Scope:  "Authority and routing", State: "Unresolved for more than 24 hours", Evidence: "Stored escalation runtime status",
			Owner: "System administration", PrimaryAction: "Review escalation routes", ActionTargetType: "CONFIGURE", ActionTargetID: "access",
			InterventionClass: today.InterventionEscalation,
		})
	}
	if count := overview.Escalation.FailedTimers; count > 0 {
		items = append(items, today.AttentionItem{
			ID: "admin-escalations-failed-timers", Type: "ESCALATION_TIMER", Title: fmt.Sprintf("Review %d failed escalation timers", count),
			WhyNow: fmt.Sprintf("%d escalation timers failed and require an administrator to restore or reroute the affected handoffs.", count),
			Scope:  "Authority and routing", State: "Timer failure", Evidence: "Stored escalation timer status",
			Owner: "System administration", PrimaryAction: "Review failed timers", ActionTargetType: "CONFIGURE", ActionTargetID: "access",
			InterventionClass: today.InterventionEscalation,
		})
	}
	return items
}

func failedJobItems(snapshot operations.Snapshot) []today.AttentionItem {
	items := make([]today.AttentionItem, 0, len(snapshot.Queues))
	for _, queue := range snapshot.Queues {
		if queue.Terminal <= 0 {
			continue
		}
		name := strings.TrimSpace(queue.Queue)
		if name == "" {
			name = "background"
		}
		items = append(items, today.AttentionItem{
			ID: "admin-jobs-" + attentionKey(name), Type: "BACKGROUND_JOB", Title: fmt.Sprintf("Review %d failed %s jobs", queue.Terminal, name),
			WhyNow: fmt.Sprintf("%d %s jobs reached a terminal failure and require an administrator to inspect recovery options.", queue.Terminal, name),
			Scope:  "System operations", State: "Terminal failure", Evidence: "Stored background job status",
			Owner: "System administration", PrimaryAction: "Review failed jobs", ActionTargetType: "CONFIGURE", ActionTargetID: "operations",
			InterventionClass: today.InterventionEscalation,
		})
	}
	return items
}

func attentionKey(values ...string) string {
	joined := strings.ToLower(strings.TrimSpace(strings.Join(values, "-")))
	joined = strings.NewReplacer(" ", "-", "_", "-", "/", "-").Replace(joined)
	joined = strings.Trim(joined, "-")
	if joined == "" {
		return "status"
	}
	return joined
}

func humanStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "Status not recorded"
	}
	return strings.ToUpper(value[:1]) + strings.ReplaceAll(value[1:], "_", " ")
}
