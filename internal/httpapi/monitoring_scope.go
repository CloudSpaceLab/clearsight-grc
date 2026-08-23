package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func (a *API) requireMonitoringProgramScope(ctx context.Context, actor identity.Actor, programID string) error {
	programID = strings.TrimSpace(programID)
	if a.deps.Continuity == nil || programID == "" {
		return monitoring.ErrNotFound
	}
	aggregate, err := a.deps.Continuity.GetProgram(ctx, actor.TenantID, programID)
	if err != nil {
		if errors.Is(err, continuity.ErrNotFound) {
			return monitoring.ErrNotFound
		}
		return err
	}
	legalEntityID := strings.TrimSpace(aggregate.Program.LegalEntityID)
	if legalEntityID == "" {
		return monitoring.ErrNotFound
	}
	if actor.LegalEntityID != "*" && strings.TrimSpace(actor.LegalEntityID) != legalEntityID {
		return monitoring.ErrNotFound
	}
	return nil
}

func (a *API) monitoringCommandPolicy(ctx context.Context, r *http.Request, tenant, name string, payload map[string]any, policy commandPolicy) (commandPolicy, error) {
	if a.deps.Monitoring == nil || a.deps.Continuity == nil {
		return policy, fmt.Errorf("monitoring or continuity service is unavailable")
	}

	programID := ""
	switch name {
	case "monitoring.check.create":
		var err error
		programID, err = boundLifecycleID(strings.TrimSpace(r.PathValue("id")), stringValue(payload["program_id"]), "program_id")
		if err != nil {
			return policy, err
		}
		policy.Responsibility = authority.ResponsibilityOwner
		policy.Materiality = max(policy.Materiality, 2)

	case "monitoring.collection.start":
		programID = stringValue(payload["program_id"])
		policy.Responsibility = authority.ResponsibilityOwner
		policy.Materiality = max(policy.Materiality, 2)

	case "monitoring.check.transition":
		version, ok := int64Value(payload["expected_version"])
		if !ok || version < 1 {
			return policy, fmt.Errorf("%w: positive expected_version is required", continuity.ErrInvalidState)
		}
		checkID, err := boundLifecycleID(strings.TrimSpace(r.PathValue("id")), stringValue(payload["id"]), "id")
		if err != nil {
			return policy, err
		}
		check, err := a.deps.Monitoring.CheckRevisionForScope(ctx, tenant, checkID, version)
		if err != nil {
			return policy, err
		}
		programID = check.ProgramID
		target := monitoring.LifecycleStatus(strings.ToUpper(stringValue(payload["to"])))
		if check.Status == monitoring.LifecyclePendingApproval && (target == monitoring.LifecycleActive || target == monitoring.LifecycleRejected) {
			policy.Responsibility = authority.ResponsibilityReviewer
			policy.Materiality = max(policy.Materiality, 3)
		} else {
			policy.Responsibility = authority.ResponsibilityOwner
			policy.Materiality = max(policy.Materiality, 2)
		}

	case "monitoring.source.evaluate":
		version, ok := int64Value(payload["check_version"])
		if !ok || version < 1 {
			return policy, fmt.Errorf("%w: positive check_version is required", continuity.ErrInvalidState)
		}
		checkID, err := boundLifecycleID(strings.TrimSpace(r.PathValue("id")), stringValue(payload["check_id"]), "check_id")
		if err != nil {
			return policy, err
		}
		check, err := a.deps.Monitoring.CheckRevisionForScope(ctx, tenant, checkID, version)
		if err != nil {
			return policy, err
		}
		programID = check.ProgramID
		policy.Responsibility = authority.ResponsibilityOwner
		policy.Materiality = max(policy.Materiality, 2)

	default:
		return policy, fmt.Errorf("%w: unsupported monitoring command", continuity.ErrInvalidState)
	}

	if strings.TrimSpace(programID) == "" {
		return policy, fmt.Errorf("%w: monitoring command is not bound to a Program", continuity.ErrInvalidState)
	}
	aggregate, err := a.deps.Continuity.GetProgram(ctx, tenant, programID)
	if err != nil {
		return policy, err
	}
	if strings.TrimSpace(aggregate.Program.LegalEntityID) == "" {
		return policy, fmt.Errorf("%w: Program has no governed legal-entity scope", continuity.ErrInvalidState)
	}

	payload["program_id"] = programID
	payload["legal_entity_id"] = aggregate.Program.LegalEntityID
	policy.ObjectIDField = "program_id"
	return policy, nil
}
