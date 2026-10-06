package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type riskIndicatorMovementRead struct {
	Direction           monitoring.MeasurementMovementDirection `json:"direction"`
	Delta               string                                  `json:"delta"`
	PreviousEvaluatedAt time.Time                               `json:"previous_evaluated_at"`
}

type riskIndicatorLabelPair struct {
	ownerID    string
	reviewerID string
}

type riskIndicatorReadBuilder struct {
	api          *API
	ctx          context.Context
	actor        identity.Actor
	monitorActor monitoring.Actor
	now          time.Time
	programs     map[string]continuity.ProgramAggregate
}

func newRiskIndicatorReadBuilder(api *API, ctx context.Context, actor identity.Actor) *riskIndicatorReadBuilder {
	return &riskIndicatorReadBuilder{
		api: api, ctx: ctx, actor: actor,
		monitorActor: monitoring.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID},
		now: time.Now().UTC(), programs: map[string]continuity.ProgramAggregate{},
	}
}

func (b *riskIndicatorReadBuilder) build(link risk.IndicatorLink) (riskIndicatorRead, riskIndicatorLabelPair, bool, bool) {
	check, err := b.api.deps.Monitoring.Check(b.ctx, b.monitorActor, link.MonitoringCheckID, link.MonitoringCheckVersion)
	if err != nil || check.ProgramID != link.ProgramID {
		return riskIndicatorRead{}, riskIndicatorLabelPair{}, false, false
	}
	program, ok := b.programs[check.ProgramID]
	if !ok {
		program, err = b.api.deps.Continuity.GetProgram(b.ctx, b.actor.TenantID, check.ProgramID)
		if err == nil {
			program, err = b.api.programForActor(b.ctx, program, nil)
		}
		if err != nil || program.Program.LegalEntityID != b.actor.LegalEntityID {
			return riskIndicatorRead{}, riskIndicatorLabelPair{}, false, false
		}
		b.programs[check.ProgramID] = program
	}

	detail := riskIndicatorRead{
		Link: link, ProgramID: program.Program.ID, ProgramName: program.Program.Name,
		CheckID: check.ID, CheckCode: check.Code, CheckName: check.Name, Claim: check.Claim,
		CheckStatus: check.Status, CheckVersion: check.Version, InputKind: check.InputKind,
		Measurement: risk.IndicatorMonitoringRiskScore, Unit: risk.IndicatorRiskScoreUnit,
		Denominator: risk.IndicatorRiskScoreDenominator, NativeMeasurement: currentRiskIndicatorNativeMeasurement(check, nil),
		State: riskIndicatorUnknown, Reason: "No current monitoring result.",
		MinimumCoverage: check.MinimumCoverage, FreshnessMinutes: check.FreshnessMinutes,
	}
	complete := true
	results, resultErr := b.api.deps.Monitoring.ListResultRevisions(b.ctx, b.monitorActor, check.ID, check.Version, 5)
	if resultErr != nil {
		complete = false
	} else if len(results) > 0 {
		current := results[0]
		detail.ResultID = current.ID
		detail.NativeMeasurement = currentRiskIndicatorNativeMeasurement(check, &current)
		detail.Score = current.Evaluation.Score
		detail.Band = current.Evaluation.Band
		coverage := current.Evaluation.Coverage
		detail.Coverage = &coverage
		evaluatedAt := current.EvaluatedAt
		detail.EvaluatedAt = &evaluatedAt
		detail.State, detail.Reason = currentRiskIndicatorState(check, current, b.now)
		if movement, ok := riskIndicatorMovement(results); ok {
			detail.Movement = &movement
		}
	}
	if program.Program.Status != continuity.ProgramActive {
		detail.State = riskIndicatorUnknown
		detail.Reason = "Source Program is not active."
	}

	episodeKey := "monitoring-check-adverse:" + check.ID
	intervention, interventionErr := b.api.deps.Continuity.OpenMatterByTriggerKey(b.ctx, b.actor.TenantID, episodeKey)
	switch {
	case interventionErr == nil:
		if riskIndicatorMatterLinkedToProgram(intervention, program.Program.ID) {
			detail.Intervention = &riskIndicatorInterventionRead{
				MatterID: intervention.Matter.ID, Reference: intervention.Matter.Reference,
				Status: intervention.Matter.Status, Priority: intervention.Matter.Priority, CreatedAt: intervention.Matter.CreatedAt,
			}
		} else {
			complete = false
		}
	case errors.Is(interventionErr, continuity.ErrNotFound):
	default:
		complete = false
	}
	return detail, riskIndicatorLabelPair{ownerID: check.OwnerPrincipalID, reviewerID: check.ReviewerPrincipalID}, true, complete
}

func (a *API) applyRiskIndicatorLabels(ctx context.Context, actor identity.Actor, details []riskIndicatorRead, pairs []riskIndicatorLabelPair) {
	if len(details) == 0 || len(details) != len(pairs) {
		return
	}
	ids := make([]string, 0, len(pairs)*2)
	for _, pair := range pairs {
		if pair.ownerID != "" {
			ids = append(ids, pair.ownerID)
		}
		if pair.reviewerID != "" {
			ids = append(ids, pair.reviewerID)
		}
	}
	labels := a.exactAssessmentLabels(ctx, actor, actor.LegalEntityID, ids)
	for index, pair := range pairs {
		details[index].OwnerDisplayName = labels[pair.ownerID]
		details[index].ReviewerDisplayName = labels[pair.reviewerID]
	}
}

func riskIndicatorMovement(results []monitoring.MonitoringResult) (riskIndicatorMovementRead, bool) {
	if len(results) < 2 || results[0].Evaluation.Measurement == nil {
		return riskIndicatorMovementRead{}, false
	}
	current := results[0]
	for _, previous := range results[1:] {
		if previous.Evaluation.Measurement == nil || sameRiskIndicatorReportingPeriod(current.Evaluation.Measurement, previous.Evaluation.Measurement) {
			continue
		}
		movement, ok := monitoring.CompareNativeMeasurements(current.Evaluation.Measurement, previous.Evaluation.Measurement)
		if !ok {
			continue
		}
		return riskIndicatorMovementRead{
			Direction: movement.Direction, Delta: movement.Delta, PreviousEvaluatedAt: previous.EvaluatedAt,
		}, true
	}
	return riskIndicatorMovementRead{}, false
}

func sameRiskIndicatorReportingPeriod(current, previous *monitoring.NativeMeasurement) bool {
	if current == nil || previous == nil {
		return false
	}
	if current.ReportingPeriodStart == nil && current.ReportingPeriodEnd == nil &&
		previous.ReportingPeriodStart == nil && previous.ReportingPeriodEnd == nil {
		return false
	}
	return equalRiskIndicatorTime(current.ReportingPeriodStart, previous.ReportingPeriodStart) &&
		equalRiskIndicatorTime(current.ReportingPeriodEnd, previous.ReportingPeriodEnd)
}

func equalRiskIndicatorTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
