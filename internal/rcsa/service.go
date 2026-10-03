package rcsa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo       Repository
	population PopulationResolver
	Now        func() time.Time
}

func NewService(repo Repository, population PopulationResolver) *Service {
	return &Service{repo: repo, population: population}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Aggregate, error) {
	if s == nil || s.repo == nil || s.population == nil {
		return Aggregate{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Aggregate{}, err
	}
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	name := strings.TrimSpace(input.Name)
	owner := strings.TrimSpace(input.FirstLineOwnerID)
	if owner == "" {
		owner = strings.TrimSpace(input.ActorID)
	}
	if code == "" || name == "" || owner == "" || !validTrigger(input.TriggerKind) {
		return Aggregate{}, ErrInvalid
	}
	riskIDs := normalizedRiskIDs(input.RiskIDs)
	if len(riskIDs) == 0 || len(riskIDs) > 100 {
		return Aggregate{}, ErrInvalid
	}
	now := s.now()
	population, err := s.population.ResolvePopulation(ctx, scope, riskIDs, now)
	if err != nil {
		return Aggregate{}, err
	}
	if err := validatePopulation(riskIDs, population); err != nil {
		return Aggregate{}, err
	}
	cycleID, err := platformid.NewUUIDv7()
	if err != nil {
		return Aggregate{}, err
	}
	for i := range population.Risks {
		population.Risks[i].CycleID = cycleID
	}
	for i := range population.Controls {
		population.Controls[i].CycleID = cycleID
	}
	checksum, err := populationChecksum(population)
	if err != nil {
		return Aggregate{}, err
	}
	cycle := Cycle{
		ID: cycleID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		Code: code, Name: name, TriggerKind: input.TriggerKind, FirstLineOwnerID: owner,
		Status: StatusDraft, PopulationChecksum: checksum, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	eventID, err := platformid.NewUUIDv7()
	if err != nil {
		return Aggregate{}, err
	}
	event := Event{
		ID: eventID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		CycleID: cycleID, CycleVersion: 1, Type: EventCycleCreated,
		ActorID: strings.TrimSpace(input.ActorID), OccurredAt: now,
	}
	return s.repo.Create(ctx, cycle, population.Risks, population.Controls, event)
}

func (s *Service) Get(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if s == nil || s.repo == nil {
		return Aggregate{}, ErrInvalid
	}
	normalized, err := normalizeScope(scope)
	if err != nil || strings.TrimSpace(id) == "" {
		return Aggregate{}, ErrInvalid
	}
	return s.repo.Get(ctx, normalized, strings.TrimSpace(id))
}

func (s *Service) ResolveLegalEntity(ctx context.Context, tenant, id string) (string, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return "", ErrInvalid
	}
	return s.repo.ResolveLegalEntity(ctx, strings.TrimSpace(tenant), strings.TrimSpace(id))
}

func normalizeScope(scope Scope) (Scope, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	if scope.TenantID == "" || scope.LegalEntityID == "" || scope.LegalEntityID == "*" {
		return Scope{}, ErrInvalid
	}
	return scope, nil
}

func validTrigger(value TriggerKind) bool {
	return value == TriggerScheduled || value == TriggerChange || value == TriggerManual
}

func normalizedRiskIDs(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func validatePopulation(requested []string, population Population) error {
	if len(population.Risks) != len(requested) {
		return ErrInvalid
	}
	seenRisks := map[string]struct{}{}
	for _, item := range population.Risks {
		if item.RiskID == "" || item.RiskVersion < 1 || item.Code == "" || item.Name == "" {
			return ErrInvalid
		}
		seenRisks[item.RiskID] = struct{}{}
	}
	for _, id := range requested {
		if _, ok := seenRisks[id]; !ok {
			return ErrInvalid
		}
	}
	seenControls := map[string]struct{}{}
	for _, item := range population.Controls {
		if _, ok := seenRisks[item.RiskID]; !ok || item.RiskVersion < 1 || item.RiskControlLinkID == "" ||
			item.CatalogLinkID == "" || item.DefinitionID == "" || item.ProgramID == "" ||
			item.ImplementationID == "" || item.ImplementationVersion < 1 || item.ImplementationName == "" ||
			item.ImplementationStatus == "" || item.ImplementationEffectiveFrom.IsZero() {
			return ErrInvalid
		}
		key := item.RiskControlLinkID
		if _, ok := seenControls[key]; ok {
			return ErrInvalid
		}
		seenControls[key] = struct{}{}
	}
	return nil
}

func populationChecksum(population Population) (string, error) {
	type canonical struct {
		Risks    []RiskSnapshot    `json:"risks"`
		Controls []ControlSnapshot `json:"controls"`
	}
	risks := append([]RiskSnapshot(nil), population.Risks...)
	controls := append([]ControlSnapshot(nil), population.Controls...)
	sort.Slice(risks, func(i, j int) bool { return risks[i].RiskID < risks[j].RiskID })
	sort.Slice(controls, func(i, j int) bool {
		if controls[i].RiskID != controls[j].RiskID {
			return controls[i].RiskID < controls[j].RiskID
		}
		return controls[i].RiskControlLinkID < controls[j].RiskControlLinkID
	})
	for i := range risks {
		risks[i].CycleID = ""
	}
	for i := range controls {
		controls[i].CycleID = ""
	}
	raw, err := json.Marshal(canonical{Risks: risks, Controls: controls})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
