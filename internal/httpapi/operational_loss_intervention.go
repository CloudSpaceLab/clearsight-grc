package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type openOperationalLossInterventionInput struct {
	ExpectedVersion int64 `json:"expected_version"`
}

func (a *API) openOperationalLossIntervention(w http.ResponseWriter, r *http.Request) {
	losses, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	if a == nil || a.deps.Continuity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "loss_intervention_unavailable", "Intervention is unavailable. Try again.")
		return
	}
	actor, scope, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	var input openOperationalLossInterventionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil || input.ExpectedVersion < 1 {
		writeOperationalLossError(w, oploss.ErrInvalid)
		return
	}
	current, err := losses.Get(r.Context(), scope, r.PathValue("id"))
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	if current.Loss.Version != input.ExpectedVersion {
		writeOperationalLossError(w, oploss.ErrVersionConflict)
		return
	}

	trusted := continuity.WithTrustedSystemEntityScope(r.Context(), actor.TenantID, actor.LegalEntityID)
	matter, err := a.ensureOperationalLossMatter(trusted, current, actor.PrincipalID)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "loss_intervention_unavailable", "Intervention is unavailable. No loss record was changed; try again.")
		return
	}
	if current.Loss.MatterID == matter.Matter.ID {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"loss": current.Loss, "matter": matter.Matter})
		return
	}

	updated, err := losses.Update(r.Context(), oploss.UpdateInput{
		TenantID: current.Loss.TenantID, LegalEntityID: current.Loss.LegalEntityID,
		LossID: current.Loss.ID, ExpectedVersion: input.ExpectedVersion,
		OrganizationScopeID: current.Loss.OrganizationScopeID,
		Title: current.Loss.Title, EventType: current.Loss.EventType, Cause: current.Loss.Cause,
		Description: current.Loss.Description, GrossAmountMinor: current.Loss.GrossAmountMinor,
		Currency: current.Loss.Currency, OccurredAt: current.Loss.OccurredAt, DiscoveredAt: current.Loss.DiscoveredAt,
		RiskID: current.Loss.RiskID, MatterID: matter.Matter.ID, Status: current.Loss.Status,
		ActorID: actor.PrincipalID,
	})
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"loss": updated, "matter": matter.Matter})
}

func (a *API) ensureOperationalLossMatter(
	ctx context.Context,
	current oploss.Aggregate,
	actorID string,
) (continuity.MatterAggregate, error) {
	if strings.TrimSpace(current.Loss.MatterID) != "" {
		matter, err := a.deps.Continuity.GetMatter(ctx, current.Loss.TenantID, current.Loss.MatterID)
		if err != nil {
			return continuity.MatterAggregate{}, err
		}
		if !operationalLossMatterMatches(current.Loss, matter.Matter) {
			return continuity.MatterAggregate{}, errors.New("linked operational loss intervention does not match loss")
		}
		return matter, nil
	}

	triggerKey := "operational-loss:" + current.Loss.ID
	matter, err := a.deps.Continuity.OpenMatterByTriggerKey(ctx, current.Loss.TenantID, triggerKey)
	if err == nil {
		if !operationalLossMatterMatches(current.Loss, matter.Matter) {
			return continuity.MatterAggregate{}, errors.New("operational loss trigger is already used by another intervention")
		}
		return matter, nil
	}
	if !errors.Is(err, continuity.ErrNotFound) {
		return continuity.MatterAggregate{}, err
	}

	summary := strings.TrimSpace(current.Loss.Description)
	if summary == "" {
		summary = current.Loss.Cause
	}
	scopeJSON, err := json.Marshal(map[string]any{
		"loss_id": current.Loss.ID,
		"currency": current.Totals.Currency,
		"gross_amount_minor": current.Totals.GrossAmountMinor,
		"recovered_amount_minor": current.Totals.RecoveredAmountMinor,
		"net_loss_minor": current.Totals.NetLossMinor,
	})
	if err != nil {
		return continuity.MatterAggregate{}, err
	}
	knownFacts, err := json.Marshal(map[string]any{
		"loss_code": current.Loss.Code,
		"event_type": current.Loss.EventType,
		"occurred_at": current.Loss.OccurredAt,
		"discovered_at": current.Loss.DiscoveredAt,
		"currency": current.Totals.Currency,
		"gross_amount_minor": current.Totals.GrossAmountMinor,
		"recovered_amount_minor": current.Totals.RecoveredAmountMinor,
		"net_loss_minor": current.Totals.NetLossMinor,
	})
	if err != nil {
		return continuity.MatterAggregate{}, err
	}

	matter, err = a.deps.Continuity.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: current.Loss.TenantID, LegalEntityID: current.Loss.LegalEntityID,
		OrganizationScopeID: current.Loss.OrganizationScopeID,
		Type: continuity.MatterOperationalLoss, Priority: 3,
		Title: current.Loss.Title, Summary: summary, Scope: scopeJSON,
		SourceType: "OPERATIONAL_LOSS", SourceID: current.Loss.ID,
		TriggerType: "MATERIAL_OPERATIONAL_LOSS", TriggerID: current.Loss.ID, TriggerKey: triggerKey,
		KnownFacts: knownFacts, MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: current.Loss.OwnerPrincipalID, RequiredAuthority: "REVIEWER", ActorID: actorID,
	})
	if errors.Is(err, continuity.ErrDuplicate) {
		matter, err = a.deps.Continuity.OpenMatterByTriggerKey(ctx, current.Loss.TenantID, triggerKey)
	}
	if err != nil {
		return continuity.MatterAggregate{}, err
	}
	if !operationalLossMatterMatches(current.Loss, matter.Matter) {
		return continuity.MatterAggregate{}, errors.New("created operational loss intervention does not match loss")
	}
	return matter, nil
}

func operationalLossMatterMatches(loss oploss.Loss, matter continuity.Matter) bool {
	return matter.TenantID == loss.TenantID &&
		matter.LegalEntityID == loss.LegalEntityID &&
		matter.Type == continuity.MatterOperationalLoss &&
		(matter.OrganizationScopeID == "" || loss.OrganizationScopeID == "" || matter.OrganizationScopeID == loss.OrganizationScopeID)
}
