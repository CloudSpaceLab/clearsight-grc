package formpolicy

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo      Repository
	forms     FormReader
	responses CompletedResponseReader
	now       func() time.Time
	newID     func() (string, error)
}

func NewService(repo Repository, forms FormReader, responses CompletedResponseReader) *Service {
	return &Service{repo: repo, forms: forms, responses: responses, now: time.Now, newID: id.NewUUIDv7}
}

func (service *Service) Create(ctx context.Context, actor Actor, input CreateInput) (Policy, error) {
	if service == nil || service.repo == nil || service.forms == nil || !validActor(actor) {
		return Policy{}, ErrInvalid
	}
	actor = normalizeActor(actor)
	now := service.currentTime()
	if err := normalizeCreateInput(&input, now); err != nil {
		return Policy{}, err
	}
	if err := service.requireActiveForm(ctx, actor, input.Eligibility); err != nil {
		return Policy{}, err
	}
	version, err := service.repo.NextPolicyVersion(ctx, actor.TenantID, actor.LegalEntityID, input.Code)
	if err != nil {
		return Policy{}, err
	}
	idValue, err := service.newID()
	if err != nil {
		return Policy{}, err
	}
	value := Policy{
		ID: idValue, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		Code: input.Code, Name: input.Name, Purpose: input.Purpose, ActionClass: ActionClassCreateMatter,
		AutomationPolicyID: input.AutomationPolicyID, AutomationPolicyVersion: input.AutomationPolicyVersion,
		Eligibility: input.Eligibility, Action: input.Action, BlastRadius: input.BlastRadius, Outcome: input.Outcome,
		Rollout: input.Rollout, Status: PolicyDraft, MakerID: actor.PrincipalID,
		EffectiveFrom: input.EffectiveFrom, EffectiveUntil: input.EffectiveUntil,
		Version: version, RecordVersion: 1, CreatedAt: now, UpdatedAt: now,
	}
	value.LastActorID = actor.PrincipalID
	value.Checksum = policyChecksum(value)
	return service.repo.CreatePolicy(ctx, value)
}

func (service *Service) Get(ctx context.Context, actor Actor, policyID string) (Policy, error) {
	if service == nil || service.repo == nil || !validActor(actor) || strings.TrimSpace(policyID) == "" {
		return Policy{}, ErrInvalid
	}
	actor = normalizeActor(actor)
	return service.repo.GetPolicy(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(policyID))
}

func (service *Service) List(ctx context.Context, actor Actor, limit int) ([]Policy, error) {
	if service == nil || service.repo == nil || !validActor(actor) || limit < 1 || limit > 200 {
		return nil, ErrInvalid
	}
	actor = normalizeActor(actor)
	return service.repo.ListPolicies(ctx, actor.TenantID, actor.LegalEntityID, limit)
}

func (service *Service) Simulate(ctx context.Context, actor Actor, policyID string, expectedVersion int64) (SimulationReceipt, error) {
	actor = normalizeActor(actor)
	value, err := service.loadForCommand(ctx, actor, policyID, expectedVersion)
	if err != nil {
		return SimulationReceipt{}, err
	}
	if value.Status != PolicyDraft && value.Status != PolicyPendingApproval && value.Status != PolicyApproved && value.Status != PolicySuspended {
		return SimulationReceipt{}, ErrInvalidTransition
	}
	if err := service.requireActiveForm(ctx, actor, value.Eligibility); err != nil {
		return SimulationReceipt{}, err
	}
	snapshot, err := service.simulatePopulation(ctx, actor, value)
	if err != nil {
		return SimulationReceipt{}, err
	}
	idValue, err := service.newID()
	if err != nil {
		return SimulationReceipt{}, err
	}
	now := service.currentTime()
	receipt := SimulationReceipt{
		ID: idValue, TenantID: value.TenantID, LegalEntityID: value.LegalEntityID, PolicyID: value.ID, PolicyVersion: value.Version,
		PolicyChecksum: value.Checksum, ActorID: actor.PrincipalID, PopulationCount: snapshot.PopulationCount, EligibleCount: snapshot.EligibleCount,
		WouldCreateCount: snapshot.WouldCreateCount, WouldReuseCount: snapshot.WouldReuseCount, BlastSuppressedCount: snapshot.BlastSuppressed,
		RestrictedExcludedCount: snapshot.RestrictedExcluded, PopulationHighWater: snapshot.HighWater, PopulationChecksum: snapshot.PopulationChecksum,
		ImpactChecksum: snapshot.ImpactChecksum, ObservedAt: now, ExpiresAt: now.Add(simulationTTL),
	}
	return service.repo.SaveSimulation(ctx, receipt)
}

func (service *Service) Submit(ctx context.Context, actor Actor, policyID string, expectedVersion int64, simulationID string) (Policy, error) {
	actor = normalizeActor(actor)
	value, err := service.loadForCommand(ctx, actor, policyID, expectedVersion)
	if err != nil {
		return Policy{}, err
	}
	if value.Status != PolicyDraft {
		return Policy{}, ErrInvalidTransition
	}
	if actor.PrincipalID != value.MakerID {
		return Policy{}, ErrMakerChecker
	}
	if _, err := service.requireFreshSimulation(ctx, actor, value, simulationID); err != nil {
		return Policy{}, err
	}
	now := service.currentTime()
	value.Status, value.ApprovedSimulationID, value.SubmittedAt = PolicyPendingApproval, strings.TrimSpace(simulationID), &now
	value.LastActorID = actor.PrincipalID
	return service.update(ctx, value, expectedVersion, now)
}

func (service *Service) Approve(ctx context.Context, actor Actor, policyID string, expectedVersion int64, simulationID string) (Policy, error) {
	actor = normalizeActor(actor)
	value, err := service.loadForCommand(ctx, actor, policyID, expectedVersion)
	if err != nil {
		return Policy{}, err
	}
	if value.Status != PolicyPendingApproval {
		return Policy{}, ErrInvalidTransition
	}
	if actor.PrincipalID == value.MakerID {
		return Policy{}, ErrMakerChecker
	}
	if strings.TrimSpace(simulationID) != value.ApprovedSimulationID {
		return Policy{}, ErrPreviewRequired
	}
	if _, err := service.requireFreshSimulation(ctx, actor, value, simulationID); err != nil {
		return Policy{}, err
	}
	now := service.currentTime()
	value.Status, value.CheckerID, value.ApprovedAt = PolicyApproved, actor.PrincipalID, &now
	value.LastActorID = actor.PrincipalID
	return service.update(ctx, value, expectedVersion, now)
}

func (service *Service) Activate(ctx context.Context, actor Actor, policyID string, expectedVersion int64) (Policy, error) {
	actor = normalizeActor(actor)
	value, err := service.loadForCommand(ctx, actor, policyID, expectedVersion)
	if err != nil {
		return Policy{}, err
	}
	if value.Status != PolicyApproved {
		return Policy{}, ErrInvalidTransition
	}
	if actor.PrincipalID == value.MakerID {
		return Policy{}, ErrMakerChecker
	}
	if err := service.requireActiveForm(ctx, actor, value.Eligibility); err != nil {
		return Policy{}, err
	}
	receipt, err := service.requireFreshSimulation(ctx, actor, value, value.ApprovedSimulationID)
	if err != nil {
		return Policy{}, err
	}
	if value.Rollout == RolloutEnforce {
		hasShadow, historyErr := service.repo.HasShadowHistory(ctx, value.TenantID, value.LegalEntityID, value.Code, value.Version)
		if historyErr != nil {
			return Policy{}, historyErr
		}
		if !hasShadow {
			return Policy{}, ErrShadowRequired
		}
	}
	current, err := service.simulatePopulation(ctx, actor, value)
	if err != nil {
		return Policy{}, err
	}
	if receipt.PopulationCount != current.PopulationCount || receipt.EligibleCount != current.EligibleCount || receipt.PopulationHighWater != current.HighWater || receipt.PopulationChecksum != current.PopulationChecksum || receipt.ImpactChecksum != current.ImpactChecksum {
		return Policy{}, ErrPreviewStale
	}
	now := service.currentTime()
	if value.EffectiveUntil != nil && !value.EffectiveUntil.After(now) {
		return Policy{}, ErrInvalidTransition
	}
	value.Status, value.CheckerID, value.ActivatedAt = PolicyActive, actor.PrincipalID, &now
	value.LastActorID = actor.PrincipalID
	return service.update(ctx, value, expectedVersion, now)
}

func (service *Service) Suspend(ctx context.Context, actor Actor, policyID string, expectedVersion int64) (Policy, error) {
	actor = normalizeActor(actor)
	value, err := service.loadForCommand(ctx, actor, policyID, expectedVersion)
	if err != nil {
		return Policy{}, err
	}
	if value.Status != PolicyActive {
		return Policy{}, ErrInvalidTransition
	}
	now := service.currentTime()
	value.Status, value.SuspendedAt = PolicySuspended, &now
	value.LastActorID = actor.PrincipalID
	return service.update(ctx, value, expectedVersion, now)
}

func (service *Service) Rollback(ctx context.Context, actor Actor, policyID string, expectedVersion int64, targetPolicyID string) (Policy, error) {
	actor = normalizeActor(actor)
	current, err := service.loadForCommand(ctx, actor, policyID, expectedVersion)
	if err != nil {
		return Policy{}, err
	}
	if current.Status != PolicyActive && current.Status != PolicySuspended {
		return Policy{}, ErrInvalidTransition
	}
	target, err := service.repo.GetPolicy(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(targetPolicyID))
	if err != nil {
		return Policy{}, err
	}
	if target.Code != current.Code || target.Version > current.Version {
		return Policy{}, ErrInvalid
	}
	if err := service.requireActiveForm(ctx, actor, target.Eligibility); err != nil {
		return Policy{}, err
	}
	version, err := service.repo.NextPolicyVersion(ctx, actor.TenantID, actor.LegalEntityID, current.Code)
	if err != nil {
		return Policy{}, err
	}
	idValue, err := service.newID()
	if err != nil {
		return Policy{}, err
	}
	now := service.currentTime()
	rolled := target
	rolled.ID, rolled.Version, rolled.RecordVersion = idValue, version, 1
	rolled.Status, rolled.MakerID, rolled.CheckerID = PolicyDraft, actor.PrincipalID, ""
	rolled.ApprovedSimulationID, rolled.SupersedesPolicyID, rolled.RollbackOfPolicyID = "", current.ID, current.ID
	rolled.SubmittedAt, rolled.ApprovedAt, rolled.ActivatedAt, rolled.SuspendedAt, rolled.RetiredAt = nil, nil, nil, nil, nil
	rolled.CreatedAt, rolled.UpdatedAt = now, now
	rolled.LastActorID = actor.PrincipalID
	rolled.Checksum = policyChecksum(rolled)
	return service.repo.CreatePolicy(ctx, rolled)
}

func (service *Service) loadForCommand(ctx context.Context, actor Actor, policyID string, expectedVersion int64) (Policy, error) {
	if service == nil || service.repo == nil || !validActor(actor) || strings.TrimSpace(policyID) == "" || expectedVersion < 1 {
		return Policy{}, ErrInvalid
	}
	value, err := service.repo.GetPolicy(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(policyID))
	if err != nil {
		return Policy{}, err
	}
	if value.RecordVersion != expectedVersion {
		return Policy{}, ErrConflict
	}
	return value, nil
}

func (service *Service) requireFreshSimulation(ctx context.Context, actor Actor, policy Policy, simulationID string) (SimulationReceipt, error) {
	if strings.TrimSpace(simulationID) == "" {
		return SimulationReceipt{}, ErrPreviewRequired
	}
	receipt, err := service.repo.GetSimulation(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(simulationID))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return SimulationReceipt{}, ErrPreviewRequired
		}
		return SimulationReceipt{}, err
	}
	now := service.currentTime()
	if receipt.PolicyID != policy.ID || receipt.PolicyVersion != policy.Version || receipt.PolicyChecksum != policy.Checksum || !receipt.ExpiresAt.After(now) || receipt.ObservedAt.After(now) {
		return SimulationReceipt{}, ErrPreviewStale
	}
	return receipt, nil
}

func (service *Service) requireActiveForm(ctx context.Context, actor Actor, eligibility Eligibility) error {
	if service == nil || service.forms == nil {
		return ErrFormInactive
	}
	form, err := service.forms.GetDistributionFormRevision(ctx, actor.TenantID, actor.LegalEntityID, eligibility.FormTemplateID, eligibility.FormTemplateVersion)
	if err != nil {
		return errors.Join(ErrFormInactive, err)
	}
	if !form.Active || form.ID != eligibility.FormTemplateID || form.Version != eligibility.FormTemplateVersion || form.TenantID != actor.TenantID || form.LegalEntityID != actor.LegalEntityID || form.ScoringMode == "" || form.ScoringMode == "NONE" {
		return ErrFormInactive
	}
	return nil
}

func (service *Service) update(ctx context.Context, value Policy, expected int64, now time.Time) (Policy, error) {
	value.RecordVersion++
	value.UpdatedAt = now
	return service.repo.UpdatePolicy(ctx, value, expected)
}

func (service *Service) currentTime() time.Time {
	if service == nil || service.now == nil {
		return time.Now().UTC()
	}
	return service.now().UTC()
}
