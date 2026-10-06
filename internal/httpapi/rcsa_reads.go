package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
)

type rcsaHandoffRead struct {
	Stage      string `json:"stage"`
	Label      string `json:"label"`
	TargetType string `json:"target_type,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
}

type rcsaCycleSummaryRead struct {
	Cycle                     rcsa.Cycle      `json:"cycle"`
	RiskCount                 int             `json:"risk_count"`
	ControlCount              int             `json:"control_count"`
	FirstLineOwnerDisplayName string          `json:"first_line_owner_display_name,omitempty"`
	Handoff                   rcsaHandoffRead `json:"handoff"`
}

type rcsaCyclePageRead struct {
	Items      []rcsaCycleSummaryRead `json:"items"`
	NextCursor string                 `json:"next_cursor,omitempty"`
	Complete   bool                   `json:"complete"`
}

type rcsaCycleDetailRead struct {
	Cycle                     rcsa.Cycle             `json:"cycle"`
	Risks                     []rcsa.RiskSnapshot    `json:"risks"`
	Controls                  []rcsa.ControlSnapshot `json:"controls"`
	FirstLineOwnerDisplayName string                 `json:"first_line_owner_display_name,omitempty"`
	AssessmentPeriodStart     *time.Time             `json:"assessment_period_start,omitempty"`
	AssessmentPeriodEnd       *time.Time             `json:"assessment_period_end,omitempty"`
	FirstLineRequestID        string                 `json:"first_line_request_id,omitempty"`
	Handoff                   rcsaHandoffRead        `json:"handoff"`
	Complete                  bool                   `json:"complete"`
}

func (a *API) listRCSACycles(w http.ResponseWriter, r *http.Request) {
	service, ok := a.rcsaService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.rcsaActorScope(w, r)
	if !ok {
		return
	}
	limit, ok := rcsaCycleListLimit(w, r)
	if !ok {
		return
	}
	page, err := service.List(r.Context(), scope, rcsa.CycleFilter{
		Status: rcsa.Status(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))),
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit:  limit,
	})
	if err != nil {
		writeRCSAError(w, err)
		return
	}

	visible, complete := a.visibleRCSACycleSummaries(r.Context(), actor, page.Items)
	ownerIDs := make([]string, 0, len(visible))
	for _, item := range visible {
		ownerIDs = append(ownerIDs, item.Cycle.FirstLineOwnerID)
	}
	labels := a.exactAssessmentLabels(r.Context(), actor, actor.LegalEntityID, ownerIDs)
	items := make([]rcsaCycleSummaryRead, 0, len(visible))
	for _, item := range visible {
		items = append(items, rcsaCycleSummaryRead{
			Cycle:                     item.Cycle,
			RiskCount:                 item.RiskCount,
			ControlCount:              item.ControlCount,
			FirstLineOwnerDisplayName: labels[item.Cycle.FirstLineOwnerID],
			Handoff:                   rcsaCycleHandoff(item.Cycle, ""),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, rcsaCyclePageRead{
		Items: items, NextCursor: page.NextCursor, Complete: complete,
	})
}

func (a *API) getRCSACycle(w http.ResponseWriter, r *http.Request) {
	service, ok := a.rcsaService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.rcsaActorScope(w, r)
	if !ok {
		return
	}
	value, err := service.Get(r.Context(), scope, r.PathValue("id"))
	if err != nil {
		writeRCSAError(w, err)
		return
	}
	if !a.canReadRCSACycle(r.Context(), actor, value.Cycle) {
		writeRCSAError(w, rcsa.ErrNotFound)
		return
	}
	labels := a.exactAssessmentLabels(r.Context(), actor, actor.LegalEntityID, []string{value.Cycle.FirstLineOwnerID})
	requestID, periodStart, periodEnd, complete := a.rcsaFirstLineContext(r.Context(), value.Cycle)
	httpx.WriteJSON(w, http.StatusOK, rcsaCycleDetailRead{
		Cycle:                     value.Cycle,
		Risks:                     value.Risks,
		Controls:                  value.Controls,
		FirstLineOwnerDisplayName: labels[value.Cycle.FirstLineOwnerID],
		AssessmentPeriodStart:     periodStart,
		AssessmentPeriodEnd:       periodEnd,
		FirstLineRequestID:        requestID,
		Handoff:                   rcsaCycleHandoff(value.Cycle, requestID),
		Complete:                  complete,
	})
}

func (a *API) visibleRCSACycleSummaries(ctx context.Context, actor identity.Actor, items []rcsa.CycleSummary) ([]rcsa.CycleSummary, bool) {
	visible := make([]rcsa.CycleSummary, 0, len(items))
	type pendingReview struct {
		item  rcsa.CycleSummary
		input authority.ResolveInput
	}
	pending := make([]pendingReview, 0)
	for _, item := range items {
		if item.Cycle.FirstLineOwnerID == actor.PrincipalID {
			visible = append(visible, item)
			continue
		}
		input, ok := rcsaReviewerReadInput(actor, item.Cycle)
		if ok {
			pending = append(pending, pendingReview{item: item, input: input})
		}
	}
	if len(pending) == 0 {
		return visible, true
	}
	resolver, ok := a.deps.Authority.(authority.BatchResolver)
	if !ok || resolver == nil {
		return visible, false
	}
	inputs := make([]authority.ResolveInput, len(pending))
	for index := range pending {
		inputs[index] = pending[index].input
	}
	outcomes, err := resolver.ResolveMany(ctx, inputs)
	if err != nil || len(outcomes) != len(pending) {
		return visible, false
	}
	complete := true
	for index, outcome := range outcomes {
		if errors.Is(outcome.Err, authority.ErrNoRoute) {
			continue
		}
		if outcome.Err != nil {
			complete = false
			continue
		}
		if outcome.Resolution.AllowsPrincipal(actor.PrincipalID) {
			visible = append(visible, pending[index].item)
		}
	}
	return visible, complete
}

func (a *API) canReadRCSACycle(ctx context.Context, actor identity.Actor, cycle rcsa.Cycle) bool {
	if cycle.FirstLineOwnerID == actor.PrincipalID {
		return true
	}
	input, ok := rcsaReviewerReadInput(actor, cycle)
	if !ok || a == nil || a.deps.Authority == nil {
		return false
	}
	resolution, err := a.deps.Authority.Resolve(ctx, input)
	return err == nil && resolution.AllowsPrincipal(actor.PrincipalID)
}

func rcsaReviewerReadInput(actor identity.Actor, cycle rcsa.Cycle) (authority.ResolveInput, bool) {
	if cycle.Status != rcsa.StatusAwaitingChallenge && cycle.Status != rcsa.StatusCompleted {
		return authority.ResolveInput{}, false
	}
	decisionType := "rcsa.challenge.start"
	if cycle.ChallengeMatterID != "" || cycle.Status == rcsa.StatusCompleted {
		decisionType = "rcsa.challenge.complete"
	}
	return authority.ResolveInput{
		TenantID: actor.TenantID, LegalEntityID: cycle.LegalEntityID,
		ObjectType: "RCSA_CYCLE", ObjectID: cycle.ID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: decisionType, Materiality: 3,
	}, true
}

func (a *API) rcsaFirstLineContext(ctx context.Context, cycle rcsa.Cycle) (string, *time.Time, *time.Time, bool) {
	if cycle.FirstLineDistributionID == "" {
		return "", nil, nil, true
	}
	if a == nil || a.deps.FormDistributions == nil || a.deps.Evidence == nil {
		return "", nil, nil, false
	}
	bundle, err := a.deps.FormDistributions.Get(ctx, cycle.TenantID, cycle.LegalEntityID, cycle.FirstLineDistributionID)
	if err != nil || bundle.Distribution.SubjectType != "RCSA_CYCLE" || bundle.Distribution.SubjectID != cycle.ID {
		return "", nil, nil, false
	}
	requestID := ""
	for _, recipient := range bundle.Recipients {
		if recipient.Role == evidence.RecipientTo &&
			recipient.Type == evidence.RecipientInternalPrincipal &&
			recipient.PrincipalID == cycle.FirstLineOwnerID &&
			recipient.State != evidence.DistributionRecipientRevoked &&
			strings.TrimSpace(recipient.RequestID) != "" {
			requestID = strings.TrimSpace(recipient.RequestID)
			break
		}
	}
	if requestID == "" {
		return "", nil, nil, false
	}
	request, err := a.deps.Evidence.GetRequest(ctx, cycle.TenantID, requestID)
	if err != nil || request.LegalEntityID != cycle.LegalEntityID ||
		request.SubjectType != "RCSA_CYCLE" || request.SubjectID != cycle.ID {
		return "", nil, nil, false
	}
	return requestID, cloneRCSATime(request.CollectionPeriodStart), cloneRCSATime(request.CollectionPeriodEnd), true
}

func rcsaCycleHandoff(cycle rcsa.Cycle, firstLineRequestID string) rcsaHandoffRead {
	switch cycle.Status {
	case rcsa.StatusDraft:
		return rcsaHandoffRead{Stage: "SETUP", Label: "First-line assessment not started"}
	case rcsa.StatusAssessmentOpen:
		value := rcsaHandoffRead{Stage: "FIRST_LINE", Label: "Complete first-line assessment"}
		if firstLineRequestID != "" {
			value.TargetType, value.TargetID = "EVIDENCE_REQUEST", firstLineRequestID
		}
		return value
	case rcsa.StatusAwaitingChallenge:
		if cycle.ChallengeMatterID == "" {
			return rcsaHandoffRead{Stage: "CHALLENGE", Label: "Start independent challenge"}
		}
		return rcsaHandoffRead{Stage: "CHALLENGE", Label: "Complete independent challenge", TargetType: "MATTER", TargetID: cycle.ChallengeMatterID}
	case rcsa.StatusCompleted:
		value := rcsaHandoffRead{Stage: "COMPLETE", Label: "Completed"}
		if cycle.ChallengeMatterID != "" {
			value.TargetType, value.TargetID = "MATTER", cycle.ChallengeMatterID
		}
		return value
	case rcsa.StatusCancelled:
		return rcsaHandoffRead{Stage: "CANCELLED", Label: "Cancelled"}
	default:
		return rcsaHandoffRead{Stage: "UNKNOWN", Label: "Status unavailable"}
	}
}

func rcsaCycleListLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 25, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "rcsa_filter_invalid", "The RCSA page size must be between 1 and 100.")
		return 0, false
	}
	return value, true
}

func cloneRCSATime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
