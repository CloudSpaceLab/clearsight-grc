package formpolicy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

const (
	simulationTTL         = 24 * time.Hour
	maximumSimulationRows = 10000
)

type populationSnapshot struct {
	PopulationCount    int
	EligibleCount      int
	WouldCreateCount   int
	WouldReuseCount    int
	BlastSuppressed    int
	RestrictedExcluded int
	HighWater          string
	PopulationChecksum string
	ImpactChecksum     string
}

func (service *Service) simulatePopulation(ctx context.Context, actor Actor, policy Policy) (populationSnapshot, error) {
	if service.responses == nil {
		return populationSnapshot{}, fmt.Errorf("%w: completed response reader is unavailable", ErrInvalid)
	}
	query := evidence.CompletedResponseQuery{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
		FormTemplateID: policy.Eligibility.FormTemplateID, FormTemplateVersion: policy.Eligibility.FormTemplateVersion,
		CurrentOnly: policy.Eligibility.CurrentOnly, Sort: evidence.ResponseSortNewest, Limit: 100,
	}
	values := make([]evidence.CompletedResponseSummary, 0)
	seen := map[string]struct{}{}
	for {
		page, err := service.responses.ListCompletedResponses(ctx, query)
		if err != nil {
			return populationSnapshot{}, err
		}
		for _, item := range page.Items {
			if _, exists := seen[item.ID]; exists {
				return populationSnapshot{}, fmt.Errorf("%w: repeated response in simulation population", ErrInvalid)
			}
			seen[item.ID] = struct{}{}
			values = append(values, item)
			if len(values) > maximumSimulationRows {
				return populationSnapshot{}, fmt.Errorf("%w: simulation population exceeds %d responses", ErrInvalid, maximumSimulationRows)
			}
		}
		if strings.TrimSpace(page.NextCursor) == "" {
			break
		}
		query.Cursor = page.NextCursor
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	snapshot := populationSnapshot{PopulationCount: len(values)}
	var highWater evidence.CompletedResponseSummary
	for _, value := range values {
		if highWater.ID == "" || value.CompletedAt.After(highWater.CompletedAt) || value.CompletedAt.Equal(highWater.CompletedAt) && value.ID > highWater.ID {
			highWater = value
		}
		if policyMatches(policy, value) {
			snapshot.EligibleCount++
		}
	}
	if highWater.ID != "" {
		snapshot.HighWater = highWater.CompletedAt.UTC().Format(time.RFC3339Nano) + "|" + highWater.ID
	}
	populationPayload, _ := json.Marshal(valuesForPopulationChecksum(values))
	populationSum := sha256.Sum256(populationPayload)
	snapshot.PopulationChecksum = hex.EncodeToString(populationSum[:])
	snapshot.WouldCreateCount = min(snapshot.EligibleCount, policy.BlastRadius.PerRun)
	snapshot.BlastSuppressed = max(0, snapshot.EligibleCount-snapshot.WouldCreateCount)
	impactPayload, _ := json.Marshal([]any{policy.Checksum, snapshot.PopulationCount, snapshot.EligibleCount, snapshot.WouldCreateCount, snapshot.WouldReuseCount, snapshot.BlastSuppressed, snapshot.RestrictedExcluded, snapshot.HighWater, snapshot.PopulationChecksum})
	impactSum := sha256.Sum256(impactPayload)
	snapshot.ImpactChecksum = hex.EncodeToString(impactSum[:])
	return snapshot, nil
}

func policyMatches(policy Policy, value evidence.CompletedResponseSummary) bool {
	eligibility := policy.Eligibility
	if value.TenantID != policy.TenantID || value.LegalEntityID != policy.LegalEntityID || value.FormTemplateID != eligibility.FormTemplateID || value.FormTemplateVersion != eligibility.FormTemplateVersion || eligibility.CurrentOnly && !value.Current || !slices.Contains(eligibility.SubjectTypes, strings.ToUpper(strings.TrimSpace(value.SubjectType))) || value.Score == nil {
		return false
	}
	score := value.Score
	if score.State != evidence.ResponseScoreFinal && score.State != evidence.ResponseScoreProvisional || score.Coverage < eligibility.MinimumCoverage {
		return false
	}
	if len(eligibility.Bands) > 0 && !slices.Contains(eligibility.Bands, score.Band) {
		return false
	}
	if eligibility.RawBelow != nil && (score.RawScore == nil || *score.RawScore >= *eligibility.RawBelow) {
		return false
	}
	if eligibility.RawAbove != nil && (score.RawScore == nil || *score.RawScore <= *eligibility.RawAbove) {
		return false
	}
	if eligibility.AdverseAtLeast != nil && (score.AdverseScore == nil || *score.AdverseScore < *eligibility.AdverseAtLeast) {
		return false
	}
	return true
}

type populationChecksumValue struct {
	ID          string                        `json:"id"`
	Current     bool                          `json:"current"`
	CompletedAt time.Time                     `json:"completed_at"`
	SubjectType string                        `json:"subject_type"`
	SubjectID   string                        `json:"subject_id"`
	Score       *evidence.ResponseScoreResult `json:"score"`
}

func valuesForPopulationChecksum(values []evidence.CompletedResponseSummary) []populationChecksumValue {
	result := make([]populationChecksumValue, 0, len(values))
	for _, value := range values {
		result = append(result, populationChecksumValue{ID: value.ID, Current: value.Current, CompletedAt: value.CompletedAt.UTC(), SubjectType: value.SubjectType, SubjectID: value.SubjectID, Score: value.Score})
	}
	return result
}
